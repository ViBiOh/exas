package exas

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	inputFilename := r.URL.Path

	if strings.Contains(inputFilename, "..") {
		httperror.BadRequest(w, errors.New("input path with dots is not allowed"))
		return
	}

	inputFilename = filepath.Join(a.workingDir, inputFilename)

	if info, err := os.Stat(inputFilename); err != nil || info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("input `%s` doesn't exist or is a directory", inputFilename))
		return
	}

	if err := a.getExif(inputFilename, w); err != nil {
		httperror.InternalServerError(w, err)
		return
	}
}
