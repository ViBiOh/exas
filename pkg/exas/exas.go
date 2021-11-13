package exas

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/streadway/amqp"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// Request for extracting exif and geocode
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

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("vith has no direct access to filesystem")
	}

	var req Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if err := checkRequest(req); err != nil {
		return err
	}

	outputFile, err := os.OpenFile(req.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to create output file: %s", err)
	}

	return a.getExif(req.Input, outputFile)
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

func checkRequest(req Request) error {
	if strings.Contains(req.Input, "..") {
		return errors.New("input path with dots is not allowed")
	}

	if strings.Contains(req.Output, "..") {
		return errors.New("output path with dots is not allowed")
	}

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	return nil
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
