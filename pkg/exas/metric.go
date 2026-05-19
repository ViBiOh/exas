package exas

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s Service) increaseMetric(ctx context.Context, source, kind, state string) {
	if s.metric == nil {
		return
	}

	s.metric.Add(ctx, 1, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("kind", kind),
		attribute.String("state", state),
	))
}

func (s Service) handleMetric(ctx context.Context, source, kind string, err error) {
	if err == nil {
		s.increaseMetric(ctx, source, kind, "success")
		return
	}

	switch {
	case errors.Is(err, errNoAccess):
		s.increaseMetric(ctx, source, kind, "no_access")
	case errors.Is(err, errUnmarshal):
		s.increaseMetric(ctx, source, kind, "unmarshal_error")
	case errors.Is(err, errExtract):
		s.increaseMetric(ctx, source, kind, "error")
	case errors.Is(err, errPublish):
		s.increaseMetric(ctx, source, kind, "publish_error")
	}
}
