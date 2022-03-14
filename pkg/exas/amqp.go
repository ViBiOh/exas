package exas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/trace"
)

type amqpResponse struct {
	Exif model.Exif `json:"exif"`
	Item absto.Item `json:"item"`
}

var (
	errNoAccess    = errors.New("exas has no direct access to filesystem")
	errUnmarshal   = errors.New("unmarshal error")
	errInvalidPath = errors.New("invalid path")
	errNotFound    = errors.New("not found")
	errExtract     = errors.New("extract error")
	errPublish     = errors.New("publish error")
)

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) (err error) {
	defer a.handleMetric("amqp", "exif", err)

	if !a.storageApp.Enabled() {
		return errNoAccess
	}

	ctx := context.Background()

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "amqp")
		defer span.End()
	}

	var item absto.Item
	if err = json.Unmarshal(message.Body, &item); err != nil {
		return fmt.Errorf("unable to decode: %s: %w", err, errUnmarshal)
	}

	reader, err := a.storageApp.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("unable to read from storage: %s", err)
	}
	defer closeWithLog(reader, "AmqpHandler", item.Pathname)

	var exif model.Exif
	exif, err = a.get(ctx, reader)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s: %w", err, errExtract)
	}

	if err = a.amqpClient.PublishJSON(amqpResponse{Item: item, Exif: exif}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("unable to publish amqp message: %s: %w", err, errPublish)
	}

	return nil
}
