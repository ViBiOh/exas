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
	geocodeService := geocode.New(config.geocode, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	exasService := exas.New(config.exas, geocodeService, clients.amqp, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	amqphandlerService, err := amqphandler.New(config.amqphandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), exasService.AmqpHandler)
	if err != nil {
		return services{}, fmt.Errorf("amqphandler: %w", err)
	}

	return services{
		exas:        exasService,
		geocode:     geocodeService,
		amqphandler: amqphandlerService,
		server:      server.New(config.server),
	}, nil
}

func (s services) Start(ctx context.Context) {
	go s.amqphandler.Start(ctx)
}

func (s services) Close() {
	s.geocode.Close()
}
