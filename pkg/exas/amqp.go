package exas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
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

func (s Service) AmqpHandler(ctx context.Context, message amqp.Delivery) (err error) {
	defer s.handleMetric(ctx, "amqp", "exif", err)

	if !s.storage.Enabled() {
		return errNoAccess
	}

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "amqp")
	defer end(&err)

	var item absto.Item
	if err = json.Unmarshal(message.Body, &item); err != nil {
		return fmt.Errorf("decode: %s: %w", err, errUnmarshal)
	}

	reader, err := s.storage.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("read from storage: %w", err)
	}
	defer closeWithLog(ctx, reader, "AmqpHandler", item.Pathname)

	var exif model.Exif
	exif, err = s.get(ctx, reader)
	if err != nil {
		return fmt.Errorf("get exif: %s: %w", err, errExtract)
	}

	if err = s.amqpClient.PublishJSON(ctx, amqpResponse{Item: item, Exif: exif}, s.amqpExchange, s.amqpRoutingKey); err != nil {
		return fmt.Errorf("publish amqp message: %s: %w", err, errPublish)
	}

	return nil
}
