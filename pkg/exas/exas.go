package exas

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ViBiOh/exas/pkg/geocode"
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
	geocodeApp geocode.App
	tmpFolder  string
	workingDir string
}

// Config of package
type Config struct {
	tmpFolder  *string
	workingDir *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder:  flags.New(prefix, "exas", "TmpFolder").Default("/tmp", overrides).Label("Folder used for temporary files storage").ToString(fs),
		workingDir: flags.New(prefix, "vith", "WorkDir").Default("", overrides).Label("Working directory for direct access requests").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, geocodeApp geocode.App) App {
	return App{
		tmpFolder:  strings.TrimSpace(*config.tmpFolder),
		workingDir: strings.TrimSpace(*config.workingDir),
		geocodeApp: geocodeApp,
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

func (a App) getExif(input string, output io.Writer) error {
	cmd := exec.Command("./exiftool", "-json", input)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to extract exif `%s`: %s", buffer.String(), err)
	}

	var exifs []map[string]interface{}
	if err := json.NewDecoder(buffer).Decode(&exifs); err != nil {
		return fmt.Errorf("unable to decode exiftool output: %s", err)
	}

	var exifData map[string]interface{}
	if len(exifs) > 0 {
		exifData = exifs[0]
	}

	if a.geocodeApp.Enabled() {
		if err := a.geocodeApp.AppendGeocoding(exifData); err != nil {
			return fmt.Errorf("unable to append geocoding: %s", err)
		}
	}

	if err := json.NewEncoder(output).Encode(exifData); err != nil {
		return fmt.Errorf("unable to marshal exif data: %s", err)
	}

	return nil
}
