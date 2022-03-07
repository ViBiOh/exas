package exas

import (
	"errors"
	"fmt"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
)

var (
	exifDates = []string{
		"SubSecCreateDate", // this one include timezone
		"CreateDate",
		"DateCreated",
	}

	exifTimezone = "OffsetTime"

	datePatterns = []string{
		"2006:01:02 15:04:05-07:00",
		"2006:01:02 15:04:05Z07:00",
		"2006:01:02 15:04:05MST",
		"01/02/2006 15:04:05-07:00",
		"01/02/2006 15:04:05Z07:00",
		"01/02/2006 15:04:05MST",
		"1/02/2006 15:04:05-07:00",
		"1/02/2006 15:04:05Z07:00",
		"1/02/2006 15:04:05MST",
	}
)

func getDate(exif model.Exif) (time.Time, error) {
	var dates []string

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

		dates = append(dates, createDateStr)
	}

	rawTimezone, ok := exif.Data[exifTimezone]
	if !ok {
		return time.Time{}, nil
	}

	timezone, ok := rawTimezone.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("key `%s` is not a string", exifTimezone)
	}

	for _, date := range dates {
		if createDate, err := parseDate(date + timezone); err == nil {
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
