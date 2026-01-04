# Trickster Codebase Guide for AI Agents

## Project Overview
Trickster is an HTTP reverse proxy cache and timeseries database (TSDB) accelerator for Prometheus, InfluxDB, ClickHouse, and IRONdb. It optimizes dashboard query performance through delta caching, step boundary normalization, and collapsed forwarding.

## Architecture

### Core Components
- **[pkg/backends/](../pkg/backends/)** - Backend abstraction layer supporting multiple TSDB providers
  - `backend.go` - Base `Backend` interface and `Registrar` pattern for HTTP handler registration
  - `timeseries_backend.go` - Extended interface for TSDB-specific operations (parsing queries, extracting time ranges)
  - Provider implementations: `prometheus/`, `influxdb/`, `clickhouse/`, `irondb/`
  
- **[pkg/proxy/](../pkg/proxy/)** - Request routing and forwarding
  - `handlers/` - HTTP handlers for caching, forwarding, health checks
  - `paths/` - Path routing and matching with per-path configuration
  - `engines/` - Query parsing and execution engines

- **[pkg/cache/](../pkg/cache/)** - Multi-backend caching (in-memory, Redis, bbolt, filesystem)
  
- **[pkg/timeseries/](../pkg/timeseries/)** - Time range query models and merging logic

### Backend Registration Pattern
All backends follow the `Registrar` pattern: `func(map[string]http.Handler)`. Each backend (e.g., Prometheus, InfluxDB) must:
1. Implement `Backend` interface
2. Call parent's `RegisterHandlers()` with a map of route names to handlers
3. Return from factory function (e.g., `NewClient()` in `prometheus/prometheus.go`)

Example from [pkg/backends/prometheus/routes.go](../pkg/backends/prometheus/routes.go):
```go
func (c *Client) RegisterHandlers(map[string]http.Handler) {
    c.TimeseriesBackend.RegisterHandlers(
        map[string]http.Handler{
            "query_range": http.HandlerFunc(c.QueryRangeHandler),
            "query":       http.HandlerFunc(c.QueryHandler),
            // ... other handlers
        },
    )
}
```

## Build & Development Workflow

### Build
```bash
make                          # fmt + build (default)
make build                    # Build current OS/ARCH
make build-linux_amd64        # Cross-compile for specific platform
make all-build                # Build all platforms (linux/amd64, linux/arm64)
```

### Test
```bash
make test                     # Runs unit-tests + e2e-tests (Docker-based)
make unit-tests               # Unit tests only: go test -race
make e2e-tests               # End-to-end tests (requires Docker, Kubernetes)
make e2e-parallel            # Parallel e2e with flake retries
```
Tests use `CGO_ENABLED=1` and vendor dependencies via `GOFLAGS="-mod=vendor"`.

### Format & Lint
```bash
make fmt                      # gofmt, goimports, shfmt
make lint                     # golangci-lint with additional checks
./hack/coverage.sh            # Coverage analysis
```

### Configuration Validation
```bash
./OPATH/trickster -validate-config -config /path/to/config.yaml
```

## Project-Specific Patterns

### 1. Time Range Query Acceleration (Core Pattern)
TSDB backends parse HTTP requests into `timeseries.TimeRangeQuery` objects capturing start/end times and step. The cache then uses these to:
- Determine which time ranges are already cached (delta proxy)
- Normalize boundaries to step intervals (step boundary normalization)
- Apply fast-forward for recent data

See [pkg/timeseries/](../pkg/timeseries/) for `TimeRangeQuery`, `RequestOptions`, and merging logic.

### 2. Handler Registration
Backends don't directly register routes; instead they provide a `RegisterHandlers()` method receiving a map. This enables:
- Dynamic handler instantiation
- Health check endpoint inclusion
- Consistent routing across all provider types

### 3. Interface-Based Extensibility
- `Backend` - Generic reverse proxy/cache interface
- `TimeseriesBackend` - TSDB-specific interface (extends `Backend`)
- Each TSDB implements both via composition: `TimeseriesBackend` embeds `Backend`

### 4. Configuration Reloading
Trickster supports graceful config reload via SIGHUP or HTTP endpoint ([docs/configuring.md](../docs/configuring.md)). Config changes don't interrupt active connections.

### 5. Provider Enumeration
Backends are registered by provider type (string) in [pkg/backends/providers/providers.go](../pkg/backends/providers/providers.go). Add new TSDB by:
1. Creating `pkg/backends/<provider>/` directory
2. Implementing `Backend` and `TimeseriesBackend` interfaces
3. Registering provider name in `Names` map and factory registration

## Key Files & Patterns

| File | Purpose |
|------|---------|
| [cmd/trickster/main.go](../cmd/trickster/main.go) | Entry point; initializes config, backends, listeners |
| [cmd/trickster/config.go](../cmd/trickster/config.go) | Config loading, validation, backend initialization |
| [pkg/backends/backend.go](../pkg/backends/backend.go) | Base `Backend` interface and common implementation |
| [pkg/backends/timeseries_backend.go](../pkg/backends/timeseries_backend.go) | `TimeseriesBackend` interface |
| [pkg/backends/prometheus/prometheus.go](../pkg/backends/prometheus/prometheus.go) | Prometheus client example |
| [pkg/proxy/handlers/](../pkg/proxy/handlers/) | HTTP request/response handlers |
| [examples/conf/example.full.yaml](../examples/conf/example.full.yaml) | Complete config documentation |

## Testing Patterns
- Unit tests: Standard Go test files (`*_test.go`) with race detector enabled
- E2E tests: Use Ginkgo framework, run in Docker container with Kubernetes support
- Mock backends available: [pkg/proxy/engines/client_test.go](../pkg/proxy/engines/client_test.go) shows `TestClient` pattern
- Test configs in [testdata/](../testdata/) (e.g., `test.*.conf` files)

## External Dependencies
- **Config**: `go.openviz.dev/trickster-config` - Configuration structs
- **Observability**: OpenTelemetry (tracing), Prometheus (metrics)
- **TSDB clients**: InfluxDB, ClickHouse SDKs, Prometheus client
- **Caching**: Redis, bbolt, Badger
- **HTTP**: Gorilla Mux, HTTP/2 support

## Conventions
- Error handling: Return wrapped errors with context (`fmt.Errorf("operation: %w", err)`)
- Logging: Use `go-kit/log` via `observability/logging`
- Metrics: Prometheus `client_golang` with standard HTTP instrument patterns
- Configuration: YAML-based with environment variable and CLI flag overrides
- Naming: Handler names map to HTTP routes (e.g., `"query_range"` → `/api/v1/query_range`)
