package exas

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/streadway/amqp"
)

// AmqpHandler for amqp request
func (a App) AmqpHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("exas has no direct access to filesystem")
	}

	inputFilename := string(message.Body)

	if strings.Contains(inputFilename, "..") {
		return errors.New("input path with dots is not allowed")
	}

	inputFilename = filepath.Join(a.workingDir, inputFilename)

	if info, err := os.Stat(inputFilename); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", inputFilename)
	}

	exif, err := a.get(inputFilename)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s", err)
	}

	payload, err := json.Marshal(exif)
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
