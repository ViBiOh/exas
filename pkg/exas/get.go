package exas

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) HandleGet(w http.ResponseWriter, r *http.Request) {
	if !s.storage.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	reader, err := s.storage.ReadFrom(ctx, r.URL.Path)
	if err != nil {
		httperror.InternalServerError(ctx, w, fmt.Errorf("read from storage: %w", err))
		return
	}
	defer closeWithLog(ctx, reader, "AmqpHandler", r.URL.Path)

	exif, err := s.get(r.Context(), reader)
	if err != nil {
		httperror.InternalServerError(ctx, w, err)
		s.increaseMetric(ctx, "http", "exif", "error")
		return
	}

	httpjson.Write(ctx, w, http.StatusOK, exif)
	s.increaseMetric(ctx, "http", "exif", "success")
}
