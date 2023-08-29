package exas

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) handleGet(w http.ResponseWriter, r *http.Request) {
	if !s.storage.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reader, err := s.storage.ReadFrom(r.Context(), r.URL.Path)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("read from storage: %w", err))
		return
	}
	defer closeWithLog(reader, "AmqpHandler", r.URL.Path)

	exif, err := s.get(r.Context(), reader)
	if err != nil {
		httperror.InternalServerError(w, err)
		s.increaseMetric(r.Context(), "http", "exif", "error")
		return
	}

	httpjson.Write(w, http.StatusOK, exif)
	s.increaseMetric(r.Context(), "http", "exif", "success")
}
