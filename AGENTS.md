# AGENTS.md

This file provides guidance to coding agents (e.g. Claude Code, claude.ai/code) when working with code in this repository.

## Repository purpose

This is the AppsCode/OpenViz fork of the CNCF [Trickster](https://github.com/trickstercache/trickster) project — an HTTP reverse proxy/cache for HTTP apps and a dashboard query accelerator for time series databases (Prometheus, ClickHouse, InfluxDB, IronDB). Produces the `trickster` binary (`v2.0.0-beta2` line).

The module path is `github.com/trickstercache/trickster/v2` (unchanged from upstream), but the repo is mirrored to `github.com/open-viz/trickster` and the Make/release/Docker harness has been rewritten to AppsCode conventions. The upstream Trickster docs under `docs/` describe behavior; everything in `hack/`, `Makefile`, `Dockerfile.in|dbg|ubi`, and the license-verifier wiring is AppsCode-side.

Note: the binary refuses to start without a valid AppsCode license — `cmd/trickster/main.go` blank-imports `go.bytebuilders.dev/license-verifier/info`.

## Architecture (Trickster 2.0)

- `cmd/trickster/` — entry point (`main.go`). Wires the runtime package, license verifier, and graceful shutdown.
- `pkg/` — Trickster core, partitioned by responsibility:
  - `backends/` — per-TSDB query handlers: `prometheus/`, `clickhouse/`, `influxdb/`, `irondb/`, plus `reverseproxy/`, `reverseproxycache/`, `rule/` (rules engine), `alb/` (Application Load Balancer), `healthcheck/`, shared `backend.go` / `backends.go` / `timeseries_backend.go`, and `providers/` / `options/` for config plumbing.
  - `cache/` — pluggable caching layer (in-memory, filesystem, bbolt, badger, Redis).
  - `frontend/`, `routing/`, `proxy/` — HTTP front door, mux, and proxy machinery.
  - `timeseries/` — generic time-series model used by all TSDB backends.
  - `parsing/` — query parser shared by backends.
  - `observability/` — Prometheus metrics, structured logging, OpenTelemetry tracing.
  - `locks/`, `checksum/`, `encoding/`, `errors/`, `runtime/`, `util/`, `testutil/` — supporting utilities.
- `docs/` — extensive Trickster user docs (ALB, caches, configuration, paths, metrics, health, negative caching, collapsed forwarding, byte range, tracing, etc.). When changing behavior, update the matching doc.
- `deploy/` — install artifacts (`helm/`, `kube/`, `systemd/`, `packaging/`).
- `examples/` — `conf/`, `docker-compose/`, `packages/` for runnable demos.
- `testdata/` — golden config files and assets used by unit tests (note the deliberately-malformed `test.bad_*.conf` cases).
- `hack/` + `Makefile` — AppsCode-style build harness (Docker-in-Docker, multi-arch, PROD/DBG/UBI image variants).
- `vendor/` — checked-in deps.

## Common commands

All build/test/lint targets run inside the AppsCode Docker build image — Docker must be running. The Makefile follows the standard AppsCode shape rather than upstream Trickster's.

- `make build` — build the binary for the host OS/ARCH into `bin/<os>_<arch>/trickster`.
- `make all-build` — build for all `BIN_PLATFORMS`.
- `make fmt` — gofmt + goimports.
- `make lint` — golangci-lint.
- `make unit-tests` — Go unit tests.
- `make e2e-tests` / `make e2e-parallel` — e2e suite.
- `make test` — `unit-tests` and `e2e-tests`.
- `make container` — build PROD (distroless), DBG (debian), and UBI image variants.
- `make push` — push all three; `make docker-manifest` writes multi-arch manifests. `make all-push` is the full publish flow.
- `make install` / `make uninstall` / `make purge` — Helm-managed install lifecycle.
- `make gen` — codegen (currently a placeholder; future generation lives here).
- `make verify` — codegen + module-tidy verification.

Run a single Go test (requires a local Go toolchain):

```
go test ./pkg/backends/prometheus/... -run TestName -v
```

For convenience, the upstream test fixtures rely on relative paths under `testdata/` — invoke tests from the package directory or use `go test ./pkg/...` from the repo root.

## Conventions

- Module path is `github.com/trickstercache/trickster/v2` (upstream); do not rename it just because the repo is hosted under `go.openviz.dev/open-viz`. Imports must use the upstream path.
- This fork is meant to **track upstream**. Prefer rebasing onto upstream Trickster changes over diverging — keep AppsCode-only additions (license-verifier wiring, AppsCode-style `Makefile`, `Dockerfile.in|dbg|ubi`, `hack/`) clearly contained so they can be replayed on top of an upstream sync.
- License: Apache-2.0 (`LICENSE`, `NOTICE`). The license header template `testdata/license_header_template.txt` is the canonical form for new files; upstream-derived files keep their original `Copyright … The Trickster Authors` header.
- Sign off commits (`git commit -s`); CONTRIBUTING.md requires DCO sign-off.
- The binary requires a valid AppsCode license at runtime via `go.bytebuilders.dev/license-verifier`. Tests do not, but anything exercising `main` does.
- Three Dockerfiles, one binary (`Dockerfile.in`, `Dockerfile.dbg`, `Dockerfile.ubi`) plus a vanilla `Dockerfile`; keep them in sync when changing build steps.
- Vendor directory is checked in — keep `go mod tidy && go mod vendor` clean.
- Project governance and maintainers: see `GOVERNANCE.md` and `MAINTAINERS.md`; security disclosures: `SECURITY.MD`.
