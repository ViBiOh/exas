package exas

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
