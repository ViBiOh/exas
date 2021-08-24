package exas

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	inputName := path.Join(a.tmpFolder, fmt.Sprintf("input_%s", sha(time.Now())))

	inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer cleanFile(inputName)

	if err := loadFile(inputFile, r); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	answerExif(w, inputName)
}

func loadFile(writer io.WriteCloser, r *http.Request) (err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}

		if closeErr := writer.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}
	}()

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, r.Body, buffer.Bytes())
	return
}
