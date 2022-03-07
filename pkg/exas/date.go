package exas

import (
	"errors"
	"fmt"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
)

var (
	exifDates = []string{
		"GPSDateTime",
		"CreateDate",
		"DateCreated",
	}

	datePatterns = []string{
		"2006:01:02 15:04:05Z07:00",
		"2006:01:02 15:04:05",
		"2006:01:02",
	}
)

func getDate(exif model.Exif) (time.Time, error) {
	for _, exifDate := range exifDates {
		rawCreateDate, ok := exif.Data[exifDate]
		if !ok {
			continue
		}

		createDateStr, ok := rawCreateDate.(string)
		if !ok {
			return time.Time{}, fmt.Errorf("key `%s` is not a string", exifDate)
		}

		if createDate, err := parseDate(createDateStr); err == nil {
			return createDate, nil
		}
	}

	return time.Time{}, nil
}

func parseDate(raw string) (time.Time, error) {
	for _, pattern := range datePatterns {
		createDate, err := time.Parse(pattern, raw)
		if err == nil {
			return createDate, nil
		}
	}

	return time.Time{}, errors.New("no matching pattern")
}
