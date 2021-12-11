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

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("exas has no direct access to filesystem")
	}

	var item model.StorageItem
	if err := json.Unmarshal(message.Body, &item); err != nil {
		a.increaseMetric("amqp", "exif", "invalid")
		return fmt.Errorf("unable to decode: %s", err)
	}

	if strings.Contains(item.Pathname, "..") {
		a.increaseMetric("amqp", "exif", "invalid_path")
		return errors.New("input path with dots is not allowed")
	}

	inputFilename := filepath.Join(a.workingDir, item.Pathname)

	if info, err := os.Stat(inputFilename); err != nil || info.IsDir() {
		a.increaseMetric("amqp", "exif", "not_found")
		return fmt.Errorf("input `%s` doesn't exist or is a directory", item.Pathname)
	}

	exif, err := a.get(inputFilename)
	if err != nil {
		a.increaseMetric("amqp", "exif", "error")
		return fmt.Errorf("unable to get exif: %s", err)
	}

	if err := a.amqpClient.PublishJSON(amqpResponse{Item: item, Exif: exif}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		a.increaseMetric("amqp", "exif", "publish_error")
		return fmt.Errorf("unable to publish amqp message: %s", err)
	}

	a.increaseMetric("amqp", "exif", "success")

	return nil
}
