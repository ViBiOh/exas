package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/exas/pkg/exas"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

type services struct {
	server      *server.Server
	amqphandler *amqphandler.Service
	exas        exas.Service
	geocode     geocode.Service
}

func newServices(config configuration, clients clients, adapters adapters) (services, error) {
	var output services
	var err error

	output.server = server.New(config.server)

	output.geocode = geocode.New(config.geocode, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	output.exas = exas.New(config.exas, output.geocode, clients.amqp, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	output.amqphandler, err = amqphandler.New(config.amqphandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.exas.AmqpHandler)
	if err != nil {
		return output, fmt.Errorf("amqphandler: %w", err)
	}

	return output, nil
}

func (s services) Start(ctx context.Context) {
	go s.amqphandler.Start(ctx)
}

func (s services) Close() {
	s.geocode.Close()
}
