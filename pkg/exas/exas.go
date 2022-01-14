package exas

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// App of package
type App struct {
	storageApp     absto.Storage
	amqpClient     *amqp.Client
	metric         *prometheus.CounterVec
	amqpExchange   string
	amqpRoutingKey string
	geocodeApp     geocode.App
}

// Config of package
type Config struct {
	amqpExchange   *string
	amqpRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		amqpExchange:   flags.New(prefix, "exas", "Exchange").Default("fibr", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey: flags.New(prefix, "exas", "RoutingKey").Default("exif_output", nil).Label("AMQP Routing Key to fibr").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, geocodeApp geocode.App, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client, storageApp absto.Storage) App {
	return App{
		geocodeApp:     geocodeApp,
		storageApp:     storageApp,
		amqpClient:     amqpClient,
		amqpExchange:   strings.TrimSpace(*config.amqpExchange),
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),

		metric: prom.CounterVec(prometheusRegisterer, "exas", "", "item", "source", "kind", "state"),
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

func (a App) get(input io.Reader) (model.Exif, error) {
	cmd := exec.Command("./exiftool", "-json", "-")

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdin = input
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
