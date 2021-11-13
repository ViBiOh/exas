package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/exas/pkg/exas"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

func main() {
	fs := flag.NewFlagSet("exas", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", "2m"), flags.NewOverride("WriteTimeout", "2m"))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))

	exasConfig := exas.Flags(fs, "")
	geocodeConfig := geocode.Flags(fs, "")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqphandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "exas"), flags.NewOverride("RoutingKey", "geocode"))

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	geocodeApp, err := geocode.New(geocodeConfig, prometheusApp.Registerer())
	logger.Fatal(err)

	exasApp := exas.New(exasConfig, geocodeApp)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer())
	if err != nil {
		logger.Error("unable to create amqp client: %s", err)
	} else {
		defer amqpClient.Close()
	}

	amqphandlerApp, err := amqphandler.New(amqphandlerConfig, amqpClient, exasApp.AmqpHandler)
	if err != nil {
		logger.Fatal(err)
	}

	go amqphandlerApp.Start(healthApp.Done())
	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(exasApp.Handler(), healthApp, recoverer.Middleware, prometheusApp.Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqphandlerApp.Done())
}
