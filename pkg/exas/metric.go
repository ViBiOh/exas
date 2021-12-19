package exas

import "errors"

func (a App) increaseMetric(source, kind, state string) {
	if a.metric == nil {
		return
	}

	a.metric.WithLabelValues(source, kind, state).Inc()
}

func (a App) handleMetric(source, kind string, err error) {
	if err == nil {
		a.increaseMetric(source, kind, "success")
		return
	}

	if errors.Is(err, errNoAccess) {
		a.increaseMetric(source, kind, "no_access")
	} else if errors.Is(err, errUnmarshal) {
		a.increaseMetric(source, kind, "unmarshal_error")
	} else if errors.Is(err, errInvalidPath) {
		a.increaseMetric(source, kind, "invalid_path")
	} else if errors.Is(err, errNotFound) {
		a.increaseMetric(source, kind, "not_found")
	} else if errors.Is(err, errExtract) {
		a.increaseMetric(source, kind, "error")
	} else if errors.Is(err, errPublish) {
		a.increaseMetric(source, kind, "publish_error")
	}
}
