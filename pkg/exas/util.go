package exas

import (
	"io"
	"log/slog"
)

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		slog.Error("close", "err", err, "fn", fn, "item", item)
	}
}
