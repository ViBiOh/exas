package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

func main() {
	fs := flag.NewFlagSet("exas", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", 2*time.Minute), flags.NewOverride("WriteTimeout", 2*time.Minute))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", uint(9090)), flags.NewOverride("IdleTimeout", 10*time.Second), flags.NewOverride("ShutdownTimeout", 5*time.Second))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	tracerConfig := tracer.Flags(fs, "tracer")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))

	exasConfig := exas.Flags(fs, "")
	abstoConfig := absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", ""))
	geocodeConfig := geocode.Flags(fs, "")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqphandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "exas"), flags.NewOverride("RoutingKey", "exif_input"))

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	ctx := context.Background()

	tracerApp, err := tracer.New(ctx, tracerConfig)
	logger.Fatal(err)
	defer tracerApp.Close(ctx)
	request.AddTracerToDefaultClient(tracerApp.GetProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	storageProvider, err := absto.New(abstoConfig, tracerApp.GetTracer("storage"))
	logger.Fatal(err)

	geocodeApp, err := geocode.New(geocodeConfig, prometheusApp.Registerer(), tracerApp.GetTracer("geocode"))
	logger.Fatal(err)
	defer geocodeApp.Close()

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer(), tracerApp.GetTracer("amqp"))
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Fatal(err)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	exasApp := exas.New(exasConfig, geocodeApp, prometheusApp.Registerer(), amqpClient, storageProvider, tracerApp.GetTracer("exas"))

	amqphandlerApp, err := amqphandler.New(amqphandlerConfig, amqpClient, tracerApp.GetTracer("amqp_handler"), exasApp.AmqpHandler)
	if err != nil {
		logger.Fatal(err)
	}

	go amqphandlerApp.Start(healthApp.ContextDone())

	go promServer.Start(healthApp.ContextEnd(), "prometheus", prometheusApp.Handler())
	go appServer.Start(healthApp.ContextEnd(), "http", httputils.Handler(exasApp.Handler(), healthApp, recoverer.Middleware, prometheusApp.Middleware, tracerApp.Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqphandlerApp.Done())
}
