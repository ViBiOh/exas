package exas

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	answerExif(w, filepath.Join(a.workingDir, r.URL.Path))
}
