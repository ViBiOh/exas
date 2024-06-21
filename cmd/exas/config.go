package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/exas/pkg/exas"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

type configuration struct {
	alcotest  *alcotest.Config
	logger    *logger.Config
	telemetry *telemetry.Config
	pprof     *pprof.Config
	server    *server.Config
	health    *health.Config

	exas        *exas.Config
	absto       *absto.Config
	geocode     *geocode.Config
	amqp        *amqp.Config
	amqphandler *amqphandler.Config
}

func newConfig() configuration {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	config := configuration{
		server:    server.Flags(fs, ""),
		health:    health.Flags(fs, ""),
		alcotest:  alcotest.Flags(fs, ""),
		logger:    logger.Flags(fs, "logger"),
		telemetry: telemetry.Flags(fs, "telemetry"),
		pprof:     pprof.Flags(fs, "pprof"),

		exas:        exas.Flags(fs, ""),
		absto:       absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", "")),
		geocode:     geocode.Flags(fs, ""),
		amqp:        amqp.Flags(fs, "amqp"),
		amqphandler: amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "exas"), flags.NewOverride("RoutingKey", "exif_input")),
	}

	_ = fs.Parse(os.Args[1:])

	return config
}
