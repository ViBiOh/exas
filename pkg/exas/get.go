package exas

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req := Request{
		Input:  filepath.Join(a.workingDir, r.URL.Path),
		Output: filepath.Join(a.workingDir, r.URL.Query().Get("output")),
	}

	if err := checkRequest(req); err != nil {
		httperror.BadRequest(w, err)
		return
	}

	outputFile, err := os.OpenFile(req.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to create output file: %s", err))
		return
	}

	if err := a.getExif(req.Input, outputFile); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
