package exas

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultExiftool   = "./exiftool"
	exiftoolWaitDelay = 5 * time.Second
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 32*1024))
	},
}

type exiftoolErrors []struct {
	Error string `json:"Error"`
}

type Service struct {
	storage        absto.Storage
	tracer         trace.Tracer
	amqpClient     *amqp.Client
	metric         metric.Int64Counter
	limiter        chan struct{}
	exiftool       string
	amqpExchange   string
	amqpRoutingKey string
	geocode        geocode.Service
}

type Config struct {
	AmqpExchange   string
	AmqpRoutingKey string
	MaxProcess     int
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("Exchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("exas").StringVar(fs, &config.AmqpExchange, "fibr", overrides)
	flags.New("RoutingKey", "AMQP Routing Key to fibr").Prefix(prefix).DocPrefix("exas").StringVar(fs, &config.AmqpRoutingKey, "exif_output", overrides)
	flags.New("MaxProcess", "Maximum number of concurrent exiftool processes").Prefix(prefix).DocPrefix("exas").IntVar(fs, &config.MaxProcess, runtime.GOMAXPROCS(0), overrides)

	return &config
}

func New(config *Config, geocodeService geocode.Service, amqpClient *amqp.Client, storageService absto.Storage, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	service := Service{
		geocode:        geocodeService,
		storage:        storageService,
		amqpClient:     amqpClient,
		limiter:        make(chan struct{}, max(config.MaxProcess, 1)),
		exiftool:       defaultExiftool,
		amqpExchange:   config.AmqpExchange,
		amqpRoutingKey: config.AmqpRoutingKey,
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/exas/pkg/exas")

		var err error

		service.metric, err = meter.Int64Counter("exas.item")
		if err != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "create counter", slog.Any("error", err))
		}
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("exas")
	}

	return service
}

func (s Service) get(ctx context.Context, input io.Reader) (exif model.Exif, err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "exiftool")
	defer end(&err)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()

	if err := s.runExiftool(ctx, input, buffer); err != nil {
		return exif, err
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

	exif.Geocode, err = s.geocode.GetGeocoding(ctx, exif)
	if err != nil {
		return exif, fmt.Errorf("append geocoding: %w", err)
	}

	return exif, nil
}

func (s Service) runExiftool(ctx context.Context, input io.Reader, output *bytes.Buffer) error {
	select {
	case s.limiter <- struct{}{}:
		defer func() { <-s.limiter }()
	case <-ctx.Done():
		return fmt.Errorf("waiting for exiftool slot: %w", ctx.Err())
	}

	cmd := exec.CommandContext(ctx, s.exiftool, "-json", "-")
	cmd.WaitDelay = exiftoolWaitDelay
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = output

	runErr := cmd.Run()
	if runErr != nil && ctx.Err() != nil {
		return fmt.Errorf("run exiftool: %w", ctx.Err())
	}

	return handleExifToolErr(runErr, output)
}

func handleExifToolErr(err error, buffer *bytes.Buffer) error {
	if err == nil {
		return nil
	}

	var toolErrs exiftoolErrors
	stderr := buffer.Bytes()

	_ = json.Unmarshal(stderr, &toolErrs)

	if len(toolErrs) > 0 && toolErrs[0].Error == "Unknown file type" {
		buffer.Reset()
		buffer.WriteString("[{}]")

		return nil
	}

	return fmt.Errorf("extract exif `%s`: %w", stderr, err)
}
