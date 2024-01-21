package exas

import (
	"context"
	"io"
	"log/slog"
)

func closeWithLog(ctx context.Context, closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "close", slog.String("fn", fn), slog.String("item", item), slog.Any("error", err))
	}
}
