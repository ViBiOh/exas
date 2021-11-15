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
		return fmt.Errorf("unable to decode: %s", err)
	}

	if strings.Contains(item.Pathname, "..") {
		return errors.New("input path with dots is not allowed")
	}

	inputFilename := filepath.Join(a.workingDir, item.Pathname)

	if info, err := os.Stat(inputFilename); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", inputFilename)
	}

	exif, err := a.get(inputFilename)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s", err)
	}

	payload, err := json.Marshal(amqpResponse{
		Item: item,
		Exif: exif,
	})
	if err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	if err = a.amqpClient.Publish(amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("unable to publish amqp message: %s", err)
	}

	return nil
}
