package exas

import (
	"io"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", fn).WithField("item", item).Error("close: %s", err)
	}
}
