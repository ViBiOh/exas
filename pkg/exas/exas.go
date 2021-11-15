package exas

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// App of package
type App struct {
	amqpClient     *amqp.Client
	tmpFolder      string
	workingDir     string
	amqpExchange   string
	amqpRoutingKey string
	geocodeApp     geocode.App
}

// Config of package
type Config struct {
	tmpFolder  *string
	workingDir *string

	amqpExchange   *string
	amqpRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder:  flags.New(prefix, "exas", "TmpFolder").Default("/tmp", overrides).Label("Folder used for temporary files storage").ToString(fs),
		workingDir: flags.New(prefix, "exas", "WorkDir").Default("", overrides).Label("Working directory for direct access requests").ToString(fs),

		amqpExchange:   flags.New(prefix, "exas", "Exchange").Default("fibr", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey: flags.New(prefix, "exas", "RoutingKey").Default("fibr", nil).Label("AMQP Routing Key for fibr").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, geocodeApp geocode.App, amqpClient *amqp.Client) App {
	return App{
		tmpFolder:  strings.TrimSpace(*config.tmpFolder),
		workingDir: strings.TrimSpace(*config.workingDir),
		geocodeApp: geocodeApp,

		amqpClient:     amqpClient,
		amqpExchange:   strings.TrimSpace(*config.amqpExchange),
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),
	}
}

// Handler for request. Should be use with net/http
func (a App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			a.handlePost(w, r)
		case http.MethodGet:
			a.handleGet(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func (a App) hasDirectAccess() bool {
	return len(a.workingDir) != 0
}

func cleanFile(name string) {
	if err := os.Remove(name); err != nil {
		logger.Warn("unable to remove file %s: %s", name, err)
	}
}

func (a App) get(input string) (model.Exif, error) {
	cmd := exec.Command("./exiftool", "-json", input)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	var exif model.Exif

	if err := cmd.Run(); err != nil {
		return exif, fmt.Errorf("unable to extract exif `%s`: %s", buffer.String(), err)
	}

	var exifs []map[string]interface{}
	if err := json.NewDecoder(buffer).Decode(&exifs); err != nil {
		return exif, fmt.Errorf("unable to decode exiftool output: %s", err)
	}

	var exifData map[string]interface{}
	if len(exifs) > 0 {
		exifData = exifs[0]
	}

	exif.Data = exifData

	if date, err := getDate(exif); err != nil {
		return exif, fmt.Errorf("unable to parse date: %s", err)
	} else if !date.IsZero() {
		exif.Date = date
	}

	if a.geocodeApp.Enabled() {
		geocode, err := a.geocodeApp.GetGeocoding(exif)
		if err != nil {
			return exif, fmt.Errorf("unable to append geocoding: %s", err)
		}

		if !geocode.IsZero() {
			exif.Geocode = geocode
		}
	}

	return exif, nil
}
