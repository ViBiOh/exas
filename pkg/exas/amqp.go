package exas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
)

const amqpHandleTimeout = time.Minute

type amqpResponse struct {
	Exif model.Exif `json:"exif"`
	Item absto.Item `json:"item"`
}

var (
	errUnmarshal = errors.New("unmarshal error")
	errRead      = errors.New("read error")
	errPublish   = errors.New("publish error")
)

func (s Service) AmqpHandler(ctx context.Context, message amqp.Delivery) error {
	err := s.handleAmqp(ctx, message)

	s.handleMetric(ctx, "amqp", err)

	if errors.Is(err, errUnmarshal) {
		slog.LogAttrs(ctx, slog.LevelError, "drop undecodable amqp message", slog.String("body", string(message.Body)), slog.Any("error", err))

		return nil
	}

	return err
}

func (s Service) handleAmqp(ctx context.Context, message amqp.Delivery) (err error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), amqpHandleTimeout)
	defer cancel()

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "amqp")
	defer end(&err)

	var item absto.Item
	if err = json.Unmarshal(message.Body, &item); err != nil {
		return fmt.Errorf("%w: decode: %w", errUnmarshal, err)
	}

	reader, err := s.storage.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("%w: read from storage: %w", errRead, err)
	}
	defer closeWithLog(ctx, reader, "AmqpHandler", item.Pathname)

	exif, err := s.get(ctx, reader)
	if err != nil {
		return fmt.Errorf("get exif: %w", err)
	}

	if err = s.amqpClient.PublishJSON(ctx, amqpResponse{Item: item, Exif: exif}, s.amqpExchange, s.amqpRoutingKey); err != nil {
		return fmt.Errorf("%w: publish amqp message: %w", errPublish, err)
	}

	return nil
}
