package exas

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer closeWithLog(ctx, r.Body, "handlePost", "input")

	exif, err := s.get(ctx, r.Body)
	if err != nil {
		s.increaseMetric(ctx, "http", "exif", "error")
		httperror.InternalServerError(ctx, w, err)
		return
	}

	s.increaseMetric(ctx, "http", "exif", "success")
	httpjson.Write(ctx, w, http.StatusOK, exif)
}
