package exas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/ViBiOh/exas/pkg/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type testEnv struct {
	service Service
	reader  *sdkmetric.ManualReader
}

func newTestEnv(t *testing.T, script string, storage absto.Storage) testEnv {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	config := &Config{Exiftool: script, MaxProcess: 2}

	return testEnv{
		service: New(config, geocode.New(&geocode.Config{}, nil, nil), nil, storage, provider, nil),
		reader:  reader,
	}
}

func (e testEnv) states(t *testing.T) map[string]int64 {
	t.Helper()

	var collected metricdata.ResourceMetrics

	require.NoError(t, e.reader.Collect(context.Background(), &collected))

	output := make(map[string]int64)

	for _, scope := range collected.ScopeMetrics {
		for _, metric := range scope.Metrics {
			sum, ok := metric.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			for _, point := range sum.DataPoints {
				state, _ := point.Attributes.Value("state")
				output[state.AsString()] += point.Value
			}
		}
	}

	return output
}

func newStorage(t *testing.T) (absto.Storage, string) {
	t.Helper()

	directory := t.TempDir()

	storage, err := filesystem.New(directory)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(directory, "photo.jpg"), []byte("content"), 0o600))

	return storage, directory
}

func TestHandlers(t *testing.T) {
	t.Parallel()

	okScript := writeScript(t, "ok", `cat >/dev/null
echo '[{"FileType":"JPEG","Thumbnail":"(Binary data 10 bytes, use -b option to extract)","CreateDate":"2020:01:02 03:04:05"}]'`)
	unknownScript := writeScript(t, "unknown", `cat >/dev/null
echo '[{"SourceFile":"-","Error":"Unknown file type"}]'
exit 1`)
	failScript := writeScript(t, "fail", `cat >/dev/null
echo "perl: broken" >&2
exit 2`)

	cases := map[string]struct {
		request     func() *http.Request
		script      string
		noStorage   bool
		wantStatus  int
		wantFileTyp string
		wantStates  map[string]int64
	}{
		"post success": {
			request:     func() *http.Request { return httptest.NewRequest(http.MethodPost, "/", strings.NewReader("content")) },
			script:      okScript,
			wantStatus:  http.StatusOK,
			wantFileTyp: "JPEG",
			wantStates:  map[string]int64{"success": 1},
		},
		"post unknown file type": {
			request:    func() *http.Request { return httptest.NewRequest(http.MethodPost, "/", strings.NewReader("content")) },
			script:     unknownScript,
			wantStatus: http.StatusOK,
			wantStates: map[string]int64{"success": 1},
		},
		"post exiftool failure": {
			request:    func() *http.Request { return httptest.NewRequest(http.MethodPost, "/", strings.NewReader("content")) },
			script:     failScript,
			wantStatus: http.StatusInternalServerError,
			wantStates: map[string]int64{"error": 1},
		},
		"get success": {
			request:     func() *http.Request { return httptest.NewRequest(http.MethodGet, "/photo.jpg", nil) },
			script:      okScript,
			wantStatus:  http.StatusOK,
			wantFileTyp: "JPEG",
			wantStates:  map[string]int64{"success": 1},
		},
		"get missing file": {
			request:    func() *http.Request { return httptest.NewRequest(http.MethodGet, "/missing.jpg", nil) },
			script:     okScript,
			wantStatus: http.StatusNotFound,
			wantStates: map[string]int64{},
		},
		"get exiftool failure": {
			request:    func() *http.Request { return httptest.NewRequest(http.MethodGet, "/photo.jpg", nil) },
			script:     failScript,
			wantStatus: http.StatusInternalServerError,
			wantStates: map[string]int64{"error": 1},
		},
		"get without storage": {
			request:    func() *http.Request { return httptest.NewRequest(http.MethodGet, "/photo.jpg", nil) },
			script:     okScript,
			noStorage:  true,
			wantStatus: http.StatusMethodNotAllowed,
			wantStates: map[string]int64{},
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			var storage absto.Storage = filesystem.Service{}
			if !testCase.noStorage {
				storage, _ = newStorage(t)
			}

			env := newTestEnv(t, testCase.script, storage)

			request := testCase.request()
			writer := httptest.NewRecorder()

			if request.Method == http.MethodPost {
				env.service.HandlePost(writer, request)
			} else {
				env.service.HandleGet(writer, request)
			}

			assert.Equal(t, testCase.wantStatus, writer.Code)
			assert.Equal(t, testCase.wantStates, env.states(t))

			if testCase.wantStatus != http.StatusOK {
				return
			}

			var got model.Exif

			require.NoError(t, json.NewDecoder(writer.Body).Decode(&got))

			if testCase.wantFileTyp != "" {
				assert.Equal(t, testCase.wantFileTyp, got.Data["FileType"])
				assert.NotContains(t, got.Data, "Thumbnail")
				assert.False(t, got.Date.IsZero())
			} else {
				assert.False(t, got.HasData())
			}
		})
	}
}

func TestAmqpHandler(t *testing.T) {
	t.Parallel()

	okScript := writeScript(t, "ok", `cat >/dev/null
echo '[{"FileType":"JPEG"}]'`)
	failScript := writeScript(t, "fail", `cat >/dev/null
exit 2`)

	cases := map[string]struct {
		body       string
		script     string
		wantErr    error
		wantStates map[string]int64
		wantFail   bool
	}{
		"success": {
			body:       `{"pathname":"/photo.jpg"}`,
			script:     okScript,
			wantStates: map[string]int64{"success": 1},
		},
		"undecodable message is dropped": {
			body:       `{`,
			script:     okScript,
			wantStates: map[string]int64{"unmarshal_error": 1},
		},
		"missing file is retried": {
			body:       `{"pathname":"/missing.jpg"}`,
			script:     okScript,
			wantErr:    errRead,
			wantStates: map[string]int64{"read_error": 1},
		},
		"extraction failure is retried": {
			body:       `{"pathname":"/photo.jpg"}`,
			script:     failScript,
			wantFail:   true,
			wantStates: map[string]int64{"error": 1},
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			storage, _ := newStorage(t)
			env := newTestEnv(t, testCase.script, storage)

			err := env.service.AmqpHandler(context.Background(), amqp.Delivery{Body: []byte(testCase.body)})

			switch {
			case testCase.wantErr != nil:
				require.ErrorIs(t, err, testCase.wantErr)
			case testCase.wantFail:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.wantStates, env.states(t))
		})
	}
}
