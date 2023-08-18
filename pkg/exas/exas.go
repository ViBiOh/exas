package exas

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// App of package
type App struct {
	storageApp     absto.Storage
	tracer         trace.Tracer
	amqpClient     *amqp.Client
	metric         metric.Int64Counter
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
		amqpExchange:   flags.New("Exchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("exas").String(fs, "fibr", overrides),
		amqpRoutingKey: flags.New("RoutingKey", "AMQP Routing Key to fibr").Prefix(prefix).DocPrefix("exas").String(fs, "exif_output", overrides),
	}
}

// New creates new App from Config
func New(config Config, geocodeApp geocode.App, meter metric.Meter, amqpClient *amqp.Client, storageApp absto.Storage, tracer trace.Tracer) App {
	var counter metric.Int64Counter
	if meter != nil {
		var err error

		counter, err = meter.Int64Counter("exas.item")
		if err != nil {
			slog.Error("create counter", "err", err)
		}
	}

	return App{
		geocodeApp:     geocodeApp,
		storageApp:     storageApp,
		amqpClient:     amqpClient,
		amqpExchange:   strings.TrimSpace(*config.amqpExchange),
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),
		tracer:         tracer,

		metric: counter,
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

func (a App) get(ctx context.Context, input io.Reader) (exif model.Exif, err error) {
	ctx, end := telemetry.StartSpan(ctx, a.tracer, "exiftool")
	defer end(&err)

	cmd := exec.Command("./exiftool", "-json", "-")

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdin = input
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return exif, fmt.Errorf("extract exif `%s`: %w", buffer.String(), err)
	}

	var exifs []map[string]any
	if err := json.NewDecoder(buffer).Decode(&exifs); err != nil {
		return exif, fmt.Errorf("decode exiftool output: %w", err)
	}

	var exifData map[string]any
	if len(exifs) > 0 {
		exifData = exifs[0]
	}

	for key, value := range exifData {
		if strValue, ok := value.(string); ok && strings.HasPrefix(strValue, "(Binary data") {
			delete(exifData, key)
		}
	}

	exif.Data = exifData
	exif.Date = getDate(exif)

	exif.Geocode, err = a.geocodeApp.GetGeocoding(ctx, exif)
	if err != nil {
		return exif, fmt.Errorf("append geocoding: %w", err)
	}

	return exif, nil
}
