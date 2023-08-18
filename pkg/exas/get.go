package exas

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if !a.storageApp.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reader, err := a.storageApp.ReadFrom(r.Context(), r.URL.Path)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("read from storage: %w", err))
		return
	}
	defer closeWithLog(reader, "AmqpHandler", r.URL.Path)

	exif, err := a.get(r.Context(), reader)
	if err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric(r.Context(), "http", "exif", "error")
		return
	}

	httpjson.Write(w, http.StatusOK, exif)
	a.increaseMetric(r.Context(), "http", "exif", "success")
}
