package exas

import (
	"fmt"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
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
		if absto.IsNotExist(err) {
			httperror.NotFound(ctx, w, nil)
			return
		}

		s.handleMetric(ctx, "http", errRead)
		httperror.InternalServerError(ctx, w, fmt.Errorf("read from storage: %w", err))

		return
	}
	defer closeWithLog(ctx, reader, "HandleGet", r.URL.Path)

	exif, err := s.get(ctx, reader)
	s.handleMetric(ctx, "http", err)

	if err != nil {
		httperror.InternalServerError(ctx, w, err)
		return
	}

	httpjson.Write(ctx, w, http.StatusOK, exif)
}
