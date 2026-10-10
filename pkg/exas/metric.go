package exas

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s Service) increaseMetric(ctx context.Context, source, state string) {
	if s.metric == nil {
		return
	}

	s.metric.Add(ctx, 1, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("kind", "exif"),
		attribute.String("state", state),
	))
}

func (s Service) handleMetric(ctx context.Context, source string, err error) {
	switch {
	case err == nil:
		s.increaseMetric(ctx, source, "success")
	case errors.Is(err, errUnmarshal):
		s.increaseMetric(ctx, source, "unmarshal_error")
	case errors.Is(err, errRead):
		s.increaseMetric(ctx, source, "read_error")
	case errors.Is(err, errPublish):
		s.increaseMetric(ctx, source, "publish_error")
	default:
		s.increaseMetric(ctx, source, "error")
	}
}
