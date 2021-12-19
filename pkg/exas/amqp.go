package exas

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/streadway/amqp"
)

type amqpResponse struct {
	Exif model.Exif        `json:"exif"`
	Item model.StorageItem `json:"item"`
}

var (
	errNoAccess    = errors.New("exas has no direct access to filesystem")
	errUnmarshal   = errors.New("unmarshal error")
	errInvalidPath = errors.New("invalid path")
	errNotFound    = errors.New("not found")
	errExtract     = errors.New("extract error")
	errPublish     = errors.New("publish error")
)

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) (err error) {
	defer a.handleMetric("amqp", "exif", err)

	if !a.hasDirectAccess() {
		return errNoAccess
	}

	var item model.StorageItem
	if err = json.Unmarshal(message.Body, &item); err != nil {
		return fmt.Errorf("unable to decode: %s: %w", err, errUnmarshal)
	}

	if strings.Contains(item.Pathname, "..") {
		return fmt.Errorf("input path with dots is not allowed: %s", errInvalidPath)
	}

	inputFilename := filepath.Join(a.workingDir, item.Pathname)

	var info os.FileInfo
	if info, err = os.Stat(inputFilename); err != nil || info.IsDir() {
		return fmt.Errorf("cannot status input `%s` or is a directory: %s", item.Pathname, errNotFound)
	}

	var exif model.Exif
	exif, err = a.get(inputFilename)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s: %w", err, errExtract)
	}

	if err = a.amqpClient.PublishJSON(amqpResponse{Item: item, Exif: exif}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("unable to publish amqp message: %s: %w", err, errPublish)
	}

	return nil
}
