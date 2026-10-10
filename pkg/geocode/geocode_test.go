package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const float64EqualityThreshold = 0.000001

func TestConvertDegreeMinuteSecondToDecimal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		location string
		wantErr  string
		want     float64
	}{
		"empty": {
			wantErr: "parse GPS data",
		},
		"north": {
			location: "1 deg 2' 3\" N",
			want:     1.034167,
		},
		"east": {
			location: "1 deg 2' 3\" E",
			want:     1.034167,
		},
		"west": {
			location: "1 deg 2' 3\" W",
			want:     -1.034167,
		},
		"south": {
			location: "1 deg 2' 3\" S",
			want:     -1.034167,
		},
		"lowercase direction": {
			location: "1 deg 2' 3\" s",
			want:     -1.034167,
		},
		"fractional seconds": {
			location: "48 deg 51' 30.36\" N",
			want:     48.8584333,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			got, err := convertDegreeMinuteSecondToDecimal(testCase.location)

			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, testCase.want, got, float64EqualityThreshold)
		})
	}
}

func TestExtractCoordinates(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		data    map[string]any
		wantErr string
		wantLat float64
		wantLon float64
	}{
		"no data": {},
		"no latitude": {
			data: map[string]any{"GPSLongitude": "2 deg 20' 0\" E"},
		},
		"empty latitude": {
			data: map[string]any{"GPSLatitude": ""},
		},
		"valid": {
			data:    map[string]any{"GPSLatitude": "48 deg 51' 30\" N", "GPSLongitude": "2 deg 21' 7\" W"},
			wantLat: 48.858333,
			wantLon: -2.351944,
		},
		"latitude not a string": {
			data:    map[string]any{"GPSLatitude": 48.8},
			wantErr: "parse latitude",
		},
		"invalid longitude": {
			data:    map[string]any{"GPSLatitude": "48 deg 51' 30\" N", "GPSLongitude": "nowhere"},
			wantErr: "parse longitude",
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			lat, lon, err := extractCoordinates(testCase.data)

			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, testCase.wantLat, lat, float64EqualityThreshold)
			assert.InDelta(t, testCase.wantLon, lon, float64EqualityThreshold)
		})
	}
}

func TestGetGeocoding(t *testing.T) {
	t.Parallel()

	gpsData := map[string]any{"GPSLatitude": "48 deg 51' 30\" N", "GPSLongitude": "2 deg 21' 7\" E"}

	cases := map[string]struct {
		data         map[string]any
		status       int
		disabled     bool
		wantErr      string
		wantRequests int32
		wantAddress  bool
	}{
		"success": {
			data:         gpsData,
			status:       http.StatusOK,
			wantRequests: 1,
			wantAddress:  true,
		},
		"no coordinates": {
			status: http.StatusOK,
		},
		"disabled keeps coordinates": {
			data:     gpsData,
			status:   http.StatusOK,
			disabled: true,
		},
		"server error": {
			data:         gpsData,
			status:       http.StatusInternalServerError,
			wantErr:      "reverse geocode",
			wantRequests: 1,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)

				assert.Equal(t, "/reverse", r.URL.Path)
				assert.Equal(t, "json", r.URL.Query().Get("format"))
				assert.Contains(t, r.UserAgent(), "exas")

				w.WriteHeader(testCase.status)
				_, _ = w.Write([]byte(`{"address":{"city":"Paris"}}`))
			}))
			t.Cleanup(server.Close)

			config := &Config{GeocodeURL: server.URL}
			if testCase.disabled {
				config.GeocodeURL = ""
			}

			service := New(config, nil, nil)
			t.Cleanup(service.Close)

			got, err := service.GetGeocoding(context.Background(), model.Exif{Data: testCase.data})

			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.wantRequests, requests.Load())
			assert.Equal(t, testCase.wantAddress, got.HasAddress())
			assert.Equal(t, testCase.data != nil, got.HasCoordinates())
		})
	}
}

func TestWaitForTurn(t *testing.T) {
	t.Parallel()

	t.Run("without ticker", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, Service{}.waitForTurn(context.Background()))
	})

	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()

		service := Service{ticker: time.NewTicker(time.Hour)}
		t.Cleanup(service.Close)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		require.ErrorIs(t, service.waitForTurn(ctx), context.DeadlineExceeded)
	})

	t.Run("tick", func(t *testing.T) {
		t.Parallel()

		service := Service{ticker: time.NewTicker(10 * time.Millisecond)}
		t.Cleanup(service.Close)

		require.NoError(t, service.waitForTurn(context.Background()))
	})
}
