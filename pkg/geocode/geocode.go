package geocode

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	gpsLatitude  = "GPSLatitude"
	gpsLongitude = "GPSLongitude"

	publicNominatimURL      = "https://nominatim.openstreetmap.org"
	publicNominatimInterval = time.Second * 2 // nominatim allows 1req/sec, so we take an extra step
)

var gpsRegex = regexp.MustCompile(`(?im)([0-9]+)\s*deg\s*([0-9]+)'\s*([0-9]+(?:\.[0-9]+)?)"\s*([N|S|W|E])`)

// Geocode content
type Geocode struct {
	Address   map[string]string `json:"address"`
	Latitude  string            `json:"lat"`
	Longitude string            `json:"lon"`
}

// App of package
type App struct {
	metric     *prometheus.CounterVec
	ticker     <-chan time.Time
	geocodeReq request.Request
}

// Config of package
type Config struct {
	geocodeURL *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		geocodeURL: flags.New(prefix, "exif", "GeocodeURL").Default("", overrides).Label(fmt.Sprintf("Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"%s\")", publicNominatimURL)).ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegisterer prometheus.Registerer) (App, error) {
	geocodeURL := strings.TrimSpace(*config.geocodeURL)

	var ticker <-chan time.Time
	if strings.HasPrefix(geocodeURL, publicNominatimURL) {
		ticker = time.Tick(publicNominatimInterval)
	}

	return App{
		geocodeReq: request.New().Header("User-Agent", "fibr, reverse geocoding from exif data").Get(geocodeURL),
		metric:     prom.CounterVec(prometheusRegisterer, "exas", "", "geocode"),
		ticker:     ticker,
	}, nil
}

// Enabled checks that requirements are met
func (a App) Enabled() bool {
	return !a.geocodeReq.IsZero()
}

// AppendGeocoding to given exif data
func (a App) AppendGeocoding(exifData map[string]interface{}) error {
	if a.ticker != nil {
		<-a.ticker
	}

	lat, lon, err := extractCoordinates(exifData)
	if err != nil {
		return fmt.Errorf("unable to get gps coordinate: %s", err)
	}

	var geocode Geocode
	if len(lat) != 0 && len(lon) != 0 {
		geocode, err = a.getReverseGeocode(context.Background(), lat, lon)
		if err != nil {
			return fmt.Errorf("unable to reverse geocode: %s", err)
		}
	}

	if len(geocode.Address) == 0 {
		a.increaseMetric("empty")
		return nil
	}

	a.increaseMetric("good")
	exifData["geocode"] = geocode

	return nil
}

func extractCoordinates(data map[string]interface{}) (string, string, error) {
	lat, err := getCoordinate(data, gpsLatitude)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse latitude: %s", err)
	}

	if len(lat) == 0 {
		return "", "", nil
	}

	lon, err := getCoordinate(data, gpsLongitude)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse longitude: %s", err)
	}

	return lat, lon, nil
}

func getCoordinate(data map[string]interface{}, key string) (string, error) {
	rawCoordinate, ok := data[key]
	if !ok {
		return "", nil
	}

	coordinateStr, ok := rawCoordinate.(string)
	if !ok {
		return "", fmt.Errorf("key `%s` is not a string", key)
	}

	coordinate, err := convertDegreeMinuteSecondToDecimal(coordinateStr)
	if err != nil {
		return "", fmt.Errorf("unable to parse `%s` with value `%s`: %s", key, coordinateStr, err)
	}

	return coordinate, nil
}

func convertDegreeMinuteSecondToDecimal(location string) (string, error) {
	matches := gpsRegex.FindAllStringSubmatch(location, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse GPS data `%s`", location)
	}

	match := matches[0]

	degrees, err := strconv.ParseFloat(match[1], 32)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS degrees: %s", err)
	}

	minutes, err := strconv.ParseFloat(match[2], 32)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS minutes: %s", err)
	}

	seconds, err := strconv.ParseFloat(match[3], 32)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS seconds: %s", err)
	}

	direction := match[4]

	dd := degrees + minutes/60.0 + seconds/3600.0

	if direction == "S" || direction == "W" {
		dd *= -1
	}

	return fmt.Sprintf("%.6f", dd), nil
}

func (a App) getReverseGeocode(ctx context.Context, lat, lon string) (Geocode, error) {
	params := url.Values{}
	params.Add("lat", lat)
	params.Add("lon", lon)
	params.Add("format", "json")
	params.Add("zoom", "18")

	a.increaseMetric("requested")

	var output Geocode

	resp, err := a.geocodeReq.Path(fmt.Sprintf("/reverse?%s", params.Encode())).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("api_error")
		return output, fmt.Errorf("unable to get reverse geocoding: %s", err)
	}

	if err = httpjson.Read(resp, &output); err != nil {
		a.increaseMetric("decode_error")
		return output, fmt.Errorf("unable to decode reverse geocoding: %s", err)
	}

	return output, nil
}

func (a App) increaseMetric(state string) {
	if a.metric == nil {
		return
	}

	a.metric.With(prometheus.Labels{
		"state": state,
	}).Inc()
}
