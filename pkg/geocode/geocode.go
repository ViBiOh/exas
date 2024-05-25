package geocode

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	gpsLatitude  = "GPSLatitude"
	gpsLongitude = "GPSLongitude"

	publicNominatimURL      = "https://nominatim.openstreetmap.org"
	publicNominatimInterval = time.Second + time.Millisecond*200 // nominatim allows 1req/sec, so we take an extra step
)

var gpsRegex = regexp.MustCompile(`(?im)([0-9]+)\s*deg\s*([0-9]+)'\s*([0-9]+(?:\.[0-9]+)?)"\s*([NSWE])`)

type reverseGeocodeResponse struct {
	Address map[string]string `json:"address"`
}

type Config struct {
	GeocodeURL string
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("GeocodeURL", fmt.Sprintf("Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"%s\")", publicNominatimURL)).Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.GeocodeURL, "", overrides)

	return &config
}

type Service struct {
	metric     metric.Int64Counter
	ticker     *time.Ticker
	tracer     trace.Tracer
	geocodeReq request.Request
}

func New(config *Config, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	var ticker *time.Ticker
	if strings.HasPrefix(config.GeocodeURL, publicNominatimURL) {
		ticker = time.NewTicker(publicNominatimInterval)
	}

	service := Service{
		geocodeReq: request.New().Header("User-Agent", "fibr, reverse geocoding from exif data").Get(config.GeocodeURL),
		ticker:     ticker,
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/exas/pkg/geocode")

		var err error

		service.metric, err = meter.Int64Counter("exas.geocode")
		if err != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "create geocode counter", slog.Any("error", err))
		}
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("geocode")
	}

	return service
}

func (s Service) Enabled() bool {
	return !s.geocodeReq.IsZero()
}

func (s Service) Close() {
	if s.ticker == nil {
		return
	}

	s.ticker.Stop()
}

func (s Service) GetGeocoding(ctx context.Context, exif model.Exif) (geocode model.Geocode, err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "geocode")
	defer end(&err)

	geocode.Latitude, geocode.Longitude, err = extractCoordinates(exif.Data)
	if err != nil {
		return geocode, fmt.Errorf("get gps coordinate: %w", err)
	}

	if !s.Enabled() {
		return
	}

	if s.ticker != nil {
		<-s.ticker.C
	}

	if geocode.HasCoordinates() {
		if geocode, err = s.getReverseGeocode(ctx, geocode); err != nil {
			return geocode, fmt.Errorf("reverse geocode: %w", err)
		}
	}

	if len(geocode.Address) == 0 {
		s.increaseMetric(ctx, "empty")
		return geocode, nil
	}

	s.increaseMetric(ctx, "success")

	return geocode, nil
}

func extractCoordinates(data map[string]any) (float64, float64, error) {
	lat, err := getCoordinate(data, gpsLatitude)
	if err != nil {
		return 0, 0, fmt.Errorf("parse latitude: %w", err)
	}

	if lat == 0 {
		return 0, 0, nil
	}

	lon, err := getCoordinate(data, gpsLongitude)
	if err != nil {
		return 0, 0, fmt.Errorf("parse longitude: %w", err)
	}

	return lat, lon, nil
}

func getCoordinate(data map[string]any, key string) (float64, error) {
	rawCoordinate, ok := data[key]
	if !ok {
		return 0, nil
	}

	coordinateStr, ok := rawCoordinate.(string)
	if !ok {
		return 0, fmt.Errorf("key `%s` is not a string", key)
	}

	if len(coordinateStr) == 0 {
		return 0, nil
	}

	coordinate, err := convertDegreeMinuteSecondToDecimal(coordinateStr)
	if err != nil {
		return 0, fmt.Errorf("parse `%s` with value `%s`: %w", key, coordinateStr, err)
	}

	return coordinate, nil
}

func convertDegreeMinuteSecondToDecimal(location string) (float64, error) {
	matches := gpsRegex.FindAllStringSubmatch(location, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("parse GPS data `%s`", location)
	}

	match := matches[0]

	degrees, err := strconv.ParseFloat(match[1], 32)
	if err != nil {
		return 0, fmt.Errorf("parse GPS degrees: %w", err)
	}

	minutes, err := strconv.ParseFloat(match[2], 32)
	if err != nil {
		return 0, fmt.Errorf("parse GPS minutes: %w", err)
	}

	seconds, err := strconv.ParseFloat(match[3], 32)
	if err != nil {
		return 0, fmt.Errorf("parse GPS seconds: %w", err)
	}

	direction := match[4]

	dd := degrees + minutes/60 + seconds/3600

	if direction == "S" || direction == "W" {
		dd *= -1
	}

	return dd, nil
}

func (s Service) getReverseGeocode(ctx context.Context, geocode model.Geocode) (model.Geocode, error) {
	params := url.Values{}
	params.Add("lat", fmt.Sprintf("%.6f", geocode.Latitude))
	params.Add("lon", fmt.Sprintf("%.6f", geocode.Longitude))
	params.Add("format", "json")
	params.Add("zoom", "18")

	s.increaseMetric(ctx, "requested")

	var reverseGeo reverseGeocodeResponse

	resp, err := s.geocodeReq.Path("/reverse?%s", params.Encode()).Send(ctx, nil)
	if err != nil {
		s.increaseMetric(ctx, "api_error")
		return geocode, fmt.Errorf("get reverse geocoding: %w", err)
	}

	if err = httpjson.Read(resp, &reverseGeo); err != nil {
		s.increaseMetric(ctx, "decode_error")
		return geocode, fmt.Errorf("decode reverse geocoding: %w", err)
	}

	geocode.Address = reverseGeo.Address

	return geocode, nil
}

func (s Service) increaseMetric(ctx context.Context, state string) {
	if s.metric == nil {
		return
	}

	s.metric.Add(ctx, 1, metric.WithAttributes(
		attribute.String("state", state),
	))
}
