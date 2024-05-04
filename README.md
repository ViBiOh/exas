# exas

[![Build](https://github.com/ViBiOh/exas/workflows/Build/badge.svg)](https://github.com/ViBiOh/exas/actions)

## API

The HTTP API is pretty simple :

- `GET /health`: healthcheck of server, always respond [`okStatus (default 204)`](#usage)
- `GET /ready`: checks external dependencies availability and then respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when close signal is received
- `GET /version`: value of `VERSION` environment variable
- `POST /`: extract Exif of the image passed in payload in binary

### Installation

Golang binary is built with static link. You can download it directly from the [GitHub Release page](https://github.com/ViBiOh/exas/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/exas](https://hub.docker.com/r/vibioh/exas/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app)

## Usage

The application can be configured by passing CLI args described below or their equivalent as environment variable. CLI values take precedence over environments variables.

Be careful when using the CLI values, if someone list the processes on the system, they will appear in plain-text. Pass secrets by environment variables: it's less easily visible.

```bash
Usage of exas:
  --address                     string    [server] Listen address ${EXAS_ADDRESS}
  --amqpExchange                string    [amqp] Exchange name ${EXAS_AMQP_EXCHANGE} (default "fibr")
  --amqpExclusive                         [amqp] Queue exclusive mode (for fanout exchange) ${EXAS_AMQP_EXCLUSIVE} (default false)
  --amqpInactiveTimeout         duration  [amqp] When inactive during the given timeout, stop listening ${EXAS_AMQP_INACTIVE_TIMEOUT} (default 0s)
  --amqpMaxRetry                uint      [amqp] Max send retries ${EXAS_AMQP_MAX_RETRY} (default 3)
  --amqpPrefetch                int       [amqp] Prefetch count for QoS ${EXAS_AMQP_PREFETCH} (default 1)
  --amqpQueue                   string    [amqp] Queue name ${EXAS_AMQP_QUEUE} (default "exas")
  --amqpRetryInterval           duration  [amqp] Interval duration when send fails ${EXAS_AMQP_RETRY_INTERVAL} (default 1h0m0s)
  --amqpRoutingKey              string    [amqp] RoutingKey name ${EXAS_AMQP_ROUTING_KEY} (default "exif_input")
  --amqpURI                     string    [amqp] Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost> ${EXAS_AMQP_URI}
  --cert                        string    [server] Certificate file ${EXAS_CERT}
  --exchange                    string    [exas] AMQP Exchange Name ${EXAS_EXCHANGE} (default "fibr")
  --geocodeURL                  string    [exif] Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. "https://nominatim.openstreetmap.org") ${EXAS_GEOCODE_URL}
  --graceDuration               duration  [http] Grace duration when signal received ${EXAS_GRACE_DURATION} (default 30s)
  --idleTimeout                 duration  [server] Idle Timeout ${EXAS_IDLE_TIMEOUT} (default 2m0s)
  --key                         string    [server] Key file ${EXAS_KEY}
  --loggerJson                            [logger] Log format as JSON ${EXAS_LOGGER_JSON} (default false)
  --loggerLevel                 string    [logger] Logger level ${EXAS_LOGGER_LEVEL} (default "INFO")
  --loggerLevelKey              string    [logger] Key for level in JSON ${EXAS_LOGGER_LEVEL_KEY} (default "level")
  --loggerMessageKey            string    [logger] Key for message in JSON ${EXAS_LOGGER_MESSAGE_KEY} (default "msg")
  --loggerTimeKey               string    [logger] Key for timestamp in JSON ${EXAS_LOGGER_TIME_KEY} (default "time")
  --name                        string    [server] Name ${EXAS_NAME} (default "http")
  --okStatus                    int       [http] Healthy HTTP Status code ${EXAS_OK_STATUS} (default 204)
  --port                        uint      [server] Listen port (0 to disable) ${EXAS_PORT} (default 1080)
  --pprofAgent                  string    [pprof] URL of the Datadog Trace Agent (e.g. http://datadog.observability:8126) ${EXAS_PPROF_AGENT}
  --readTimeout                 duration  [server] Read Timeout ${EXAS_READ_TIMEOUT} (default 2m0s)
  --routingKey                  string    [exas] AMQP Routing Key to fibr ${EXAS_ROUTING_KEY} (default "exif_output")
  --shutdownTimeout             duration  [server] Shutdown Timeout ${EXAS_SHUTDOWN_TIMEOUT} (default 10s)
  --storageFileSystemDirectory  /data     [storage] Path to directory. Default is dynamic. /data on a server and Current Working Directory in a terminal. ${EXAS_STORAGE_FILE_SYSTEM_DIRECTORY}
  --storageObjectAccessKey      string    [storage] Storage Object Access Key ${EXAS_STORAGE_OBJECT_ACCESS_KEY}
  --storageObjectBucket         string    [storage] Storage Object Bucket ${EXAS_STORAGE_OBJECT_BUCKET}
  --storageObjectClass          string    [storage] Storage Object Class ${EXAS_STORAGE_OBJECT_CLASS}
  --storageObjectEndpoint       string    [storage] Storage Object endpoint ${EXAS_STORAGE_OBJECT_ENDPOINT}
  --storageObjectRegion         string    [storage] Storage Object Region ${EXAS_STORAGE_OBJECT_REGION}
  --storageObjectSSL                      [storage] Use SSL ${EXAS_STORAGE_OBJECT_SSL} (default true)
  --storageObjectSecretAccess   string    [storage] Storage Object Secret Access ${EXAS_STORAGE_OBJECT_SECRET_ACCESS}
  --storagePartSize             uint      [storage] PartSize configuration ${EXAS_STORAGE_PART_SIZE} (default 5242880)
  --telemetryRate               string    [telemetry] OpenTelemetry sample rate, 'always', 'never' or a float value ${EXAS_TELEMETRY_RATE} (default "always")
  --telemetryURL                string    [telemetry] OpenTelemetry gRPC endpoint (e.g. otel-exporter:4317) ${EXAS_TELEMETRY_URL}
  --telemetryUint64                       [telemetry] Change OpenTelemetry Trace ID format to an unsigned int 64 ${EXAS_TELEMETRY_UINT64} (default true)
  --url                         string    [alcotest] URL to check ${EXAS_URL}
  --userAgent                   string    [alcotest] User-Agent for check ${EXAS_USER_AGENT} (default "Alcotest")
  --writeTimeout                duration  [server] Write Timeout ${EXAS_WRITE_TIMEOUT} (default 2m0s)
```
