package exas

import (
	"context"
	"io"
	"log/slog"
)

func closeWithLog(ctx context.Context, closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		slog.ErrorContext(ctx, "close", "error", err, "fn", fn, "item", item)
	}
}
