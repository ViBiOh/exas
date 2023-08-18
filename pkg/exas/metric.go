package exas

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (a App) increaseMetric(ctx context.Context, source, kind, state string) {
	if a.metric == nil {
		return
	}

	a.metric.Add(ctx, 1, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("kind", kind),
		attribute.String("state", state),
	))
}

func (a App) handleMetric(ctx context.Context, source, kind string, err error) {
	if err == nil {
		a.increaseMetric(ctx, source, kind, "success")
		return
	}

	if errors.Is(err, errNoAccess) {
		a.increaseMetric(ctx, source, kind, "no_access")
	} else if errors.Is(err, errUnmarshal) {
		a.increaseMetric(ctx, source, kind, "unmarshal_error")
	} else if errors.Is(err, errInvalidPath) {
		a.increaseMetric(ctx, source, kind, "invalid_path")
	} else if errors.Is(err, errNotFound) {
		a.increaseMetric(ctx, source, kind, "not_found")
	} else if errors.Is(err, errExtract) {
		a.increaseMetric(ctx, source, kind, "error")
	} else if errors.Is(err, errPublish) {
		a.increaseMetric(ctx, source, kind, "publish_error")
	}
}
