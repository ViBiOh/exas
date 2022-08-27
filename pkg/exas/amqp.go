package exas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/streadway/amqp"
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
func (a App) AmqpHandler(ctx context.Context, message amqp.Delivery) (err error) {
	defer a.handleMetric("amqp", "exif", err)

	if !a.storageApp.Enabled() {
		return errNoAccess
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end()

	var item absto.Item
	if err = json.Unmarshal(message.Body, &item); err != nil {
		return fmt.Errorf("decode: %s: %w", err, errUnmarshal)
	}

	reader, err := a.storageApp.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("read from storage: %w", err)
	}
	defer closeWithLog(reader, "AmqpHandler", item.Pathname)

	var exif model.Exif
	exif, err = a.get(ctx, reader)
	if err != nil {
		return fmt.Errorf("get exif: %s: %w", err, errExtract)
	}

	if err = a.amqpClient.PublishJSON(amqpResponse{Item: item, Exif: exif}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("publish amqp message: %s: %w", err, errPublish)
	}

	return nil
}
