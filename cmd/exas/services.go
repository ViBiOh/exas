package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
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

	amqpClient := clients.amqp
	if _, err := adapters.storage.List(context.Background(), "/"); errors.Is(err, absto.ErrNotEnabled) && amqpClient != nil {
		slog.LogAttrs(context.Background(), slog.LevelWarn, "no storage configured, amqp messages are not consumed")

		amqpClient = nil
	}

	output.amqphandler, err = amqphandler.New(config.amqphandler, amqpClient, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.exas.AmqpHandler)
	if err != nil {
		return output, fmt.Errorf("amqphandler: %w", err)
	}

	return output, nil
}

func (s services) Start(ctx context.Context) {
	s.amqphandler.Start(ctx)
}

func (s services) Close() {
	s.geocode.Close()
}
