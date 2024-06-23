package main

import (
	"context"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

func main() {
	config := newConfig()
	alcotest.DoAndExit(config.alcotest)

	ctx := context.Background()

	clients, err := newClients(ctx, config)
	logger.FatalfOnErr(ctx, err, "clients")

	defer clients.Close(ctx)
	go clients.Start()

	adapters, err := newAdapters(config, clients)
	logger.FatalfOnErr(ctx, err, "adapters")

	services, err := newServices(config, clients, adapters)
	logger.FatalfOnErr(ctx, err, "services")

	defer services.Close()
	go services.Start(clients.health.DoneCtx())

	port := newPort(services)

	go services.server.Start(clients.health.EndCtx(), httputils.Handler(port, clients.health, clients.telemetry.Middleware("http")))

	clients.health.WaitForTermination(services.server.Done())

	server.GracefulWait(services.server.Done(), services.amqphandler.Done())
}
