# CLAUDE.md — gost/

CLI binary entry point for GOST (GO Simple Tunnel). This module compiles the `gost` command.

## Build & Run

```bash
cd gost && go build ./cmd/gost/...

# Cross-compile
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" ./cmd/gost/...

# Build all platforms
make all

# Run
./gost -L "http://:8080" -F "socks5://:1080"
./gost -C gost.yml
```

## Structure

| File | Purpose |
|------|---------|
| `cmd/gost/main.go` | CLI entry point: flag parsing, multi-worker mode (`--` separator), program launch via `svc.Run` |
| `cmd/gost/program.go` | Service lifecycle (`Init`/`Start`/`Stop`), config loading, API/metrics/profiling servers, SIGHUP reload |
| `cmd/gost/register.go` | Blank imports for all built-in handlers, listeners, dialers, connectors — triggers `init()` registration |
| `cmd/gost/version.go` | Version string (`3.3.0`) |

## CLI Flags

| Flag | Usage |
|------|-------|
| `-L` | Inline service definition (repeatable) |
| `-F` | Inline chain node definition (repeatable) |
| `-C` | Path to config file (YAML/JSON) |
| `-D` / `-DD` | Debug / trace logging |
| `-api` | API service address (e.g. `:8080`) |
| `-metrics` | Prometheus metrics address |
| `-O` | Output merged config as yaml or json, then exit |
| `-V` | Print version and exit |

## Multi-Worker Mode

Arguments separated by ` -- ` spawn multiple worker processes via `exec.CommandContext`. Each worker runs as a child process with `_GOST_ID` set. Any worker's exit cancels all others. This is triggered in `init()` before flag parsing.

## Service Lifecycle

- `Init` → calls `parser.Init()` with all CLI/config inputs
- `Start` → `parser.Parse()` → `loader.Load()` → `p.run()` (starts services, API, metrics, profiling)
- `Stop` → cancels reload context, closes all services
- SIGHUP → `reloadConfig()` re-parses and re-runs without restarting the process

## Key Dependencies

- `github.com/go-gost/core` — interface definitions
- `github.com/go-gost/x` — all implementations, config, registry
- `github.com/judwhite/go-svc` — OS service framework (handles daemon/SIGHUP/SIGTERM)

## Registration Pattern

All components register via `init()` side-effects in their packages. `cmd/gost/register.go` triggers them with blank imports. When adding a new built-in handler/listener/dialer/connector, add a blank import here.

## Tests

Tests are in `tests/e2e/` — integration tests using the built binary. No unit tests. Run with `go test ./tests/e2e/...`.
