package exas

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) HandlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer closeWithLog(ctx, r.Body, "handlePost", "input")

	exif, err := s.get(ctx, r.Body)
	s.handleMetric(ctx, "http", err)

	if err != nil {
		httperror.InternalServerError(ctx, w, err)
		return
	}

	httpjson.Write(ctx, w, http.StatusOK, exif)
}
