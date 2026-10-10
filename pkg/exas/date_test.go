package exas

import (
	"testing"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestGetDate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		data map[string]any
		want time.Time
	}{
		"no data": {},
		"invalid date": {
			data: map[string]any{"DateTimeOriginal": "0000:00:00 00:00:00"},
		},
		"not a string": {
			data: map[string]any{"DateTimeOriginal": 2020},
		},
		"date without timezone is UTC": {
			data: map[string]any{"DateTimeOriginal": "2020:01:02 03:04:05"},
			want: time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC),
		},
		"date only": {
			data: map[string]any{"CreateDate": "2020:01:02"},
			want: time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		"embedded timezone": {
			data: map[string]any{"GPSDateTime": "2020:01:02 03:04:05Z"},
			want: time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC),
		},
		"offset time": {
			data: map[string]any{"DateTimeOriginal": "2020:01:02 03:04:05", "OffsetTime": "+02:00"},
			want: time.Date(2020, time.January, 2, 1, 4, 5, 0, time.UTC),
		},
		"sub seconds with timezone": {
			data: map[string]any{"SubSecDateTimeOriginal": "2020:01:02 03:04:05.123+02:00"},
			want: time.Date(2020, time.January, 2, 1, 4, 5, 123000000, time.UTC),
		},
		"gps date has priority": {
			data: map[string]any{"DateTimeOriginal": "2020:01:02 03:04:05", "GPSDateTime": "2021:02:03 04:05:06Z"},
			want: time.Date(2021, time.February, 3, 4, 5, 6, 0, time.UTC),
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			got := getDate(model.Exif{Data: testCase.data})

			assert.True(t, testCase.want.Equal(got), "got %s, want %s", got, testCase.want)
		})
	}
}
