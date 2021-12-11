package exas

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	inputName := path.Join(a.tmpFolder, fmt.Sprintf("input_%s", sha.New(time.Now())))

	writer, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "exif", "not_found")
		return
	}

	defer cleanFile(inputName)

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			logger.WithField("fn", "vith.handlePost").WithField("item", inputName).Error("unable to close: %s", err)
		}
	}()

	if err := loadFile(writer, r); err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "exif", "load_error")
		return
	}

	exif, err := a.get(inputName)
	if err != nil {
		a.increaseMetric("http", "exif", "error")
		httperror.InternalServerError(w, err)
		return
	}

	a.increaseMetric("http", "exif", "success")
	httpjson.Write(w, http.StatusOK, exif)
}

func loadFile(writer io.Writer, r *http.Request) (err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			} else {
				err = fmt.Errorf("unable to close: %s", err)
			}
		}
	}()

	_, err = io.Copy(writer, r.Body)
	return
}
