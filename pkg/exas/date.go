package exas

import (
	"time"

	"github.com/ViBiOh/exas/pkg/model"
)

var (
	offsetTimeName = "OffsetTime"

	exifDates = []string{
		"GPSDateTime",
		"DateCreated",
		"DateTimeOriginal",
		"SubSecCreateDate",
		"CreateDate",
	}

	tzPattern = "2006:01:02 15:04:05Z07:00"

	datePatterns = []string{
		"2006:01:02 15:04:05",
		"2006:01:02",
	}
)

func getDate(exif model.Exif) time.Time {
	var dates []string

	for _, exifDate := range exifDates {
		if date := getExifString(exif, exifDate); len(date) > 0 {
			dates = append(dates, date)
		}
	}

	if createDate := parseDateWithTimezone(exif, dates); !createDate.IsZero() {
		return createDate
	}

	for _, pattern := range datePatterns {
		for _, date := range dates {
			if createDate, err := time.Parse(pattern, date); err == nil {
				return createDate
			}
		}
	}

	return time.Time{}
}

func parseDateWithTimezone(exif model.Exif, dates []string) time.Time {
	offsetTime := getExifString(exif, offsetTimeName)

	for _, date := range dates {
		if createDate, err := time.Parse(tzPattern, date); err == nil {
			return createDate
		}

		if len(offsetTime) > 0 {
			if createDate, err := time.Parse(tzPattern, date+offsetTime); err == nil {
				return createDate
			}
		}
	}

	return time.Time{}
}

func getExifString(exif model.Exif, key string) string {
	rawDate, ok := exif.Data[key]
	if !ok {
		return ""
	}

	date, ok := rawDate.(string)
	if ok {
		return date
	}

	return ""
}
