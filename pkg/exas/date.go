package exas

import (
	"errors"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
)

var (
	exifDates = []string{
		"GPSDateTime",
		"SubSecCreateDate",
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
	var dates []string

	for _, exifDate := range exifDates {
		rawDate, ok := exif.Data[exifDate]
		if !ok {
			continue
		}

		date, ok := rawDate.(string)
		if ok {
			dates = append(dates, date)
		}
	}

	for _, pattern := range datePatterns {
		for _, date := range dates {
			if createDate, err := time.Parse(pattern, date); err == nil {
				return createDate, nil
			}
		}
	}

	return time.Time{}, errors.New("no matching pattern")
}
