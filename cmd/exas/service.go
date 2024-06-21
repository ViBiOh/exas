package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/exas/pkg/exas"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

type service struct {
	amqphandler *amqphandler.Service
	exas        exas.Service
	geocode     geocode.Service
	server      server.Server
}

func newService(config configuration, clients clients, adapters adapters) (*service, error) {
	geocodeService := geocode.New(config.geocode, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	exasService := exas.New(config.exas, geocodeService, clients.amqp, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	amqphandlerService, err := amqphandler.New(config.amqphandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), exasService.AmqpHandler)
	if err != nil {
		return nil, fmt.Errorf("amqphandler: %w", err)
	}

	return &service{
		exas:        exasService,
		geocode:     geocodeService,
		amqphandler: amqphandlerService,
		server:      server.New(config.server),
	}, nil
}

func (s *service) Start(ctx context.Context) {
	s.amqphandler.Start(ctx)
}

func (s *service) Close() {
	s.geocode.Close()
}
