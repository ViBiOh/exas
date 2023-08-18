package exas

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	defer closeWithLog(r.Body, "handlePost", "input")

	exif, err := a.get(r.Context(), r.Body)
	if err != nil {
		a.increaseMetric(r.Context(), "http", "exif", "error")
		httperror.InternalServerError(w, err)
		return
	}

	a.increaseMetric(r.Context(), "http", "exif", "success")
	httpjson.Write(w, http.StatusOK, exif)
}
