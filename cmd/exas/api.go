package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/exas/pkg/exas"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

func main() {
	fs := flag.NewFlagSet("exas", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", 2*time.Minute), flags.NewOverride("WriteTimeout", 2*time.Minute))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	telemetryConfig := telemetry.Flags(fs, "telemetry")

	exasConfig := exas.Flags(fs, "")
	abstoConfig := absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", ""))
	geocodeConfig := geocode.Flags(fs, "")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqphandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "exas"), flags.NewOverride("RoutingKey", "exif_input"))

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	alcotest.DoAndExit(alcotestConfig)

	logger.Init(loggerConfig)

	ctx := context.Background()

	telemetryService, err := telemetry.New(ctx, telemetryConfig)
	if err != nil {
		slog.Error("create telemetry", "err", err)
		os.Exit(1)
	}

	defer telemetryService.Close(ctx)
	request.AddOpenTelemetryToDefaultClient(telemetryService.MeterProvider(), telemetryService.TracerProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	healthService := health.New(healthConfig)

	storageProvider, err := absto.New(abstoConfig, telemetryService.TracerProvider())
	if err != nil {
		slog.Error("create absto", "err", err)
		os.Exit(1)
	}

	geocodeService := geocode.New(geocodeConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	defer geocodeService.Close()

	amqpClient, err := amqp.New(amqpConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		slog.Error("create amqp", "err", err)
		os.Exit(1)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	exasService := exas.New(exasConfig, geocodeService, amqpClient, storageProvider, telemetryService.MeterProvider(), telemetryService.TracerProvider())

	amqphandlerService, err := amqphandler.New(amqphandlerConfig, amqpClient, telemetryService.MeterProvider(), telemetryService.TracerProvider(), exasService.AmqpHandler)
	if err != nil {
		slog.Error("create amqp handler", "err", err)
		os.Exit(1)
	}

	go amqphandlerService.Start(healthService.Done(ctx))

	endCtx := healthService.End(ctx)

	go appServer.Start(endCtx, "http", httputils.Handler(exasService.Handler(), healthService, recoverer.Middleware, telemetryService.Middleware("http")))

	healthService.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), amqphandlerService.Done())
}
