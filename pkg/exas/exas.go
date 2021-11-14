package exas

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/streadway/amqp"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// Request for generating exif file
type Request struct {
	Input  string `json:"input"`
	Output string `json:"output"`
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
		workingDir: flags.New(prefix, "exas", "WorkDir").Default("", overrides).Label("Working directory for direct access requests").ToString(fs),
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

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("exas has no direct access to filesystem")
	}

	var req Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if strings.Contains(req.Input, "..") {
		return errors.New("input path with dots is not allowed")
	}

	if strings.Contains(req.Output, "..") {
		return errors.New("output path with dots is not allowed")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	exif, err := a.get(req.Input)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s", err)
	}

	writer, err := os.OpenFile(req.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open output file: %s", err)
	}

	if err = json.NewEncoder(writer).Encode(exif); err != nil {
		err = fmt.Errorf("unable to encode: %s", err)
	}

	if closeErr := writer.Close(); closeErr != nil {
		if err != nil {
			return fmt.Errorf("%s: %w", err, closeErr)
		}
		return fmt.Errorf("unable to close writer: %s", closeErr)
	}

	return err
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
