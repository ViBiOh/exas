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

	if errors.Is(err, errNoAccess) {
		s.increaseMetric(ctx, source, kind, "no_access")
	} else if errors.Is(err, errUnmarshal) {
		s.increaseMetric(ctx, source, kind, "unmarshal_error")
	} else if errors.Is(err, errInvalidPath) {
		s.increaseMetric(ctx, source, kind, "invalid_path")
	} else if errors.Is(err, errNotFound) {
		s.increaseMetric(ctx, source, kind, "not_found")
	} else if errors.Is(err, errExtract) {
		s.increaseMetric(ctx, source, kind, "error")
	} else if errors.Is(err, errPublish) {
		s.increaseMetric(ctx, source, kind, "publish_error")
	}
}
