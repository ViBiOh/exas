package exas

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) handlePost(w http.ResponseWriter, r *http.Request) {
	defer closeWithLog(r.Body, "handlePost", "input")

	exif, err := s.get(r.Context(), r.Body)
	if err != nil {
		s.increaseMetric(r.Context(), "http", "exif", "error")
		httperror.InternalServerError(w, err)
		return
	}

	s.increaseMetric(r.Context(), "http", "exif", "success")
	httpjson.Write(w, http.StatusOK, exif)
}
