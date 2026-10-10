package exas

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViBiOh/exas/pkg/geocode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeScript(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+content+"\n"), 0o700))

	return path
}

func TestRunExiftool(t *testing.T) {
	t.Parallel()

	// Scripts are written before any subtest runs in parallel, so that no
	// writable file descriptor is open while another test forks (ETXTBSY).
	okScript := writeScript(t, "ok", `echo '[{"FileType":"JPEG"}]'`)
	hangScript := writeScript(t, "hang", `exec sleep 30`)

	cases := map[string]struct {
		prefill  bool
		script   string
		timeout  time.Duration
		wantErr  error
		wantBody string
	}{
		"success": {
			script:   okScript,
			timeout:  time.Minute,
			wantBody: `[{"FileType":"JPEG"}]`,
		},
		"cancelled while running": {
			script:  hangScript,
			timeout: 200 * time.Millisecond,
			wantErr: context.DeadlineExceeded,
		},
		"cancelled while waiting for a slot": {
			prefill: true,
			script:  okScript,
			timeout: 200 * time.Millisecond,
			wantErr: context.DeadlineExceeded,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service := Service{
				limiter:  make(chan struct{}, 1),
				exiftool: testCase.script,
			}

			if testCase.prefill {
				service.limiter <- struct{}{}
			}

			ctx, cancel := context.WithTimeout(context.Background(), testCase.timeout)
			defer cancel()

			var output bytes.Buffer

			start := time.Now()
			err := service.runExiftool(ctx, strings.NewReader("input"), &output)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				assert.Less(t, time.Since(start), exiftoolWaitDelay)

				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, testCase.wantBody, output.String())
			assert.Len(t, service.limiter, 0)
		})
	}
}

func TestRunExiftoolLimiter(t *testing.T) {
	t.Parallel()

	// The script fails when another instance holds the lock directory, which
	// happens only if two processes run at the same time.
	lockDir := filepath.Join(t.TempDir(), "lock")
	script := writeScript(t, "exclusive", `mkdir "`+lockDir+`" || exit 3
sleep 0.1
rmdir "`+lockDir+`"
echo '[{}]'`)

	service := Service{
		limiter:  make(chan struct{}, 1),
		exiftool: script,
	}

	const callers = 5

	errs := make(chan error, callers)

	var wg sync.WaitGroup

	for range callers {
		wg.Go(func() {
			var output bytes.Buffer

			errs <- service.runExiftool(context.Background(), strings.NewReader("input"), &output)
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
}

func TestHandleExifToolErr(t *testing.T) {
	t.Parallel()

	runErr := errors.New("exit status 1")

	cases := map[string]struct {
		err        error
		stdout     string
		stderr     string
		wantErr    string
		wantStdout string
	}{
		"no error": {
			stdout:     `[{"a":"b"}]`,
			wantStdout: `[{"a":"b"}]`,
		},
		"unknown file type": {
			err:        runErr,
			stdout:     `[{"SourceFile":"-","Error":"Unknown file type"}]`,
			wantStdout: "[{}]",
		},
		"other exiftool error": {
			err:        runErr,
			stdout:     `[{"SourceFile":"-","Error":"File is empty"}]`,
			wantErr:    "File is empty",
			wantStdout: `[{"SourceFile":"-","Error":"File is empty"}]`,
		},
		"stderr is preferred": {
			err:        runErr,
			stdout:     `not json`,
			stderr:     "perl: broken",
			wantErr:    "perl: broken",
			wantStdout: `not json`,
		},
		"stdout is the fallback": {
			err:        runErr,
			stdout:     `not json`,
			wantErr:    "not json",
			wantStdout: `not json`,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			stdout := bytes.NewBufferString(testCase.stdout)

			err := handleExifToolErr(testCase.err, stdout, []byte(testCase.stderr))

			if testCase.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, runErr)
				assert.ErrorContains(t, err, testCase.wantErr)
			}

			assert.Equal(t, testCase.wantStdout, stdout.String())
		})
	}
}

func TestGetGeocodeFailure(t *testing.T) {
	t.Parallel()

	script := writeScript(t, "gps", `cat >/dev/null
echo '[{"GPSLatitude":"48 deg 51'"'"' 30\" N","GPSLongitude":"2 deg 21'"'"' 7\" E"}]'`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	geocoder := geocode.New(&geocode.Config{GeocodeURL: server.URL}, nil, nil)
	t.Cleanup(geocoder.Close)

	service := New(&Config{Exiftool: script, MaxProcess: 1}, geocoder, nil, nil, nil, nil)

	t.Run("keeps exif and coordinates", func(t *testing.T) {
		t.Parallel()

		got, err := service.get(context.Background(), strings.NewReader("input"))

		require.NoError(t, err)
		assert.True(t, got.HasData())
		assert.True(t, got.Geocode.HasCoordinates())
		assert.False(t, got.Geocode.HasAddress())
	})

	t.Run("fails when cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := service.get(ctx, strings.NewReader("input"))

		require.ErrorIs(t, err, context.Canceled)
	})
}
