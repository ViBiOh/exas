# exas

[![Build](https://github.com/ViBiOh/exas/workflows/Build/badge.svg)](https://github.com/ViBiOh/exas/actions)
[![codecov](https://codecov.io/gh/ViBiOh/exas/branch/main/graph/badge.svg)](https://codecov.io/gh/ViBiOh/exas)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_exas&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_exas)

## API

The HTTP API is pretty simple :

- `GET /health`: healthcheck of server, always respond [`okStatus (default 204)`](#usage)
- `GET /ready`: checks external dependencies availability and then respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when `SIGTERM` is received
- `GET /version`: value of `VERSION` environment variable
- `GET /metrics`: Prometheus metrics, on a dedicated port [`prometheusPort (default 9090)`](#usage)
- `POST /`: extract Exif of the image passed in payload in binary

### Installation

Golang binary is built with static link. You can download it directly from the [Github Release page](https://github.com/ViBiOh/exas/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/exas](https://hub.docker.com/r/vibioh/exas/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app)

## CI

Following variables are required for CI:

|      Name       |           Purpose           |
| :-------------: | :-------------------------: |
| **DOCKER_USER** | for publishing Docker image |
| **DOCKER_PASS** | for publishing Docker image |

## Usage

The application can be configured by passing CLI args described below or their equivalent as environment variable. CLI values take precedence over environments variables.

Be careful when using the CLI values, if someone list the processes on the system, they will appear in plain-text. Pass secrets by environment variables: it's less easily visible.

```bash
Usage of exas:
  -address string
        [server] Listen address {EXAS_ADDRESS}
  -cert string
        [server] Certificate file {EXAS_CERT}
  -geocodeURL string
        [exif] Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. "https://nominatim.openstreetmap.org") {EXAS_GEOCODE_URL}
  -graceDuration string
        [http] Grace duration when SIGTERM received {EXAS_GRACE_DURATION} (default "30s")
  -idleTimeout string
        [server] Idle Timeout {EXAS_IDLE_TIMEOUT} (default "2m")
  -key string
        [server] Key file {EXAS_KEY}
  -loggerJson
        [logger] Log format as JSON {EXAS_LOGGER_JSON}
  -loggerLevel string
        [logger] Logger level {EXAS_LOGGER_LEVEL} (default "INFO")
  -loggerLevelKey string
        [logger] Key for level in JSON {EXAS_LOGGER_LEVEL_KEY} (default "level")
  -loggerMessageKey string
        [logger] Key for message in JSON {EXAS_LOGGER_MESSAGE_KEY} (default "message")
  -loggerTimeKey string
        [logger] Key for timestamp in JSON {EXAS_LOGGER_TIME_KEY} (default "time")
  -okStatus int
        [http] Healthy HTTP Status code {EXAS_OK_STATUS} (default 204)
  -port uint
        [server] Listen port (0 to disable) {EXAS_PORT} (default 1080)
  -prometheusAddress string
        [prometheus] Listen address {EXAS_PROMETHEUS_ADDRESS}
  -prometheusCert string
        [prometheus] Certificate file {EXAS_PROMETHEUS_CERT}
  -prometheusGzip
        [prometheus] Enable gzip compression of metrics output {EXAS_PROMETHEUS_GZIP}
  -prometheusIdleTimeout string
        [prometheus] Idle Timeout {EXAS_PROMETHEUS_IDLE_TIMEOUT} (default "10s")
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {EXAS_PROMETHEUS_IGNORE}
  -prometheusKey string
        [prometheus] Key file {EXAS_PROMETHEUS_KEY}
  -prometheusPort uint
        [prometheus] Listen port (0 to disable) {EXAS_PROMETHEUS_PORT} (default 9090)
  -prometheusReadTimeout string
        [prometheus] Read Timeout {EXAS_PROMETHEUS_READ_TIMEOUT} (default "5s")
  -prometheusShutdownTimeout string
        [prometheus] Shutdown Timeout {EXAS_PROMETHEUS_SHUTDOWN_TIMEOUT} (default "5s")
  -prometheusWriteTimeout string
        [prometheus] Write Timeout {EXAS_PROMETHEUS_WRITE_TIMEOUT} (default "10s")
  -readTimeout string
        [server] Read Timeout {EXAS_READ_TIMEOUT} (default "2m")
  -shutdownTimeout string
        [server] Shutdown Timeout {EXAS_SHUTDOWN_TIMEOUT} (default "10s")
  -tmpFolder string
        [exas] Folder used for temporary files storage {EXAS_TMP_FOLDER} (default "/tmp")
  -url string
        [alcotest] URL to check {EXAS_URL}
  -userAgent string
        [alcotest] User-Agent for check {EXAS_USER_AGENT} (default "Alcotest")
  -workDir string
        [vith] Working directory for direct access requests {EXAS_WORK_DIR}
  -writeTimeout string
        [server] Write Timeout {EXAS_WRITE_TIMEOUT} (default "2m")
```
