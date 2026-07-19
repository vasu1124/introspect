# AGENTS.md - Introspect Project Agent Guidelines

This document provides context and guidelines for AI agents working on the Introspect project.

---

## 📌 Project Overview

**Introspect** is a Go-based Kubernetes web application demonstrating cloud-native patterns:
- Leader election with Kubernetes leases
- Kubernetes operators (controller-runtime)
- Admission webhooks
- Database integrations (MongoDB, etcd, Valkey/Redis)
- Mandelbrot fractal HPA simulation
- Prometheus metrics, health checks, structured logging

**Tech Stack**: Go 1.26, Kubernetes client-go, controller-runtime, Cobra, Viper, Zap, Prometheus client

---

## 📁 Project Structure

```
introspect/
├── cmd/
│   └── introspect/
│       ├── main.go       # Entry point
│       ├── root.go       # Cobra root command & config (Viper)
│       └── server.go     # Server subcommand
├── pkg/
│   ├── assets/           # Embedded templates (HTML) & static assets (CSS)
│   ├── config/           # Centralized config structs
│   ├── handler/          # Handler interface definition
│   ├── middleware/       # HTTP middleware (request logging)
│   ├── server/           # HTTP server with handler registry
│   ├── logger/           # Zap-based structured logging (logr)
│   ├── version/          # Build-time version info
│   ├── signal/           # Signal handling for graceful shutdown
│   │
│   ├── environ/          # Environment introspection endpoint
│   ├── cookie/           # Cookie management demo
│   ├── dynconfig/        # Dynamic config reload (fsnotify + Viper)
│   ├── validate/         # Admission webhook + UI for pod validation
│   ├── election/         # Kubernetes leader election demo
│   ├── guestbook/        # Multi-backend guestbook (MongoDB, etcd, Valkey)
│   ├── mandelbrot/       # Mandelbrot fractal HPA demo + Prometheus metrics
│   ├── healthz/          # Health/readiness endpoints
│   ├── operator/         # Kubernetes operator (controller-runtime) + WebSocket
│   ├── network/          # Network info
│   └── osinfo/           # OS info (build tags: linux/darwin)
├── kubernetes/           # K8s manifests (kapp/kustomize)
├── tmpl/                 # Source HTML templates (served from filesystem)
├── css/                  # Source CSS/JS (served from filesystem)
├── etc/config/           # Example config files
├── hack/                 # Build scripts (TLS certs, etc.)
├── Tiltfile              # Tilt dev workflow
├── Makefile              # Build automation
└── Dockerfile            # Multi-stage build
```

---

## 🎯 Key Architectural Patterns

### Handler Interface (pkg/handler/handler.go)
All feature packages implement `handler.Handler`:
```go
type Handler interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux, ctx context.Context)
}
type Closer interface { Close() error }  // Optional cleanup
```
Registered via `server.BuildHandlers()` → `server.RegisterHandlers()` → `Server.Run()`.

### Server (pkg/server/server.go)
- Creates `http.ServeMux`
- Registers static routes (`/`, `/metrics`, `/css/`, `/favicon.ico`)
- Delegates feature routes to registered handlers
- Applies middleware chain (request logging)
- Graceful shutdown with context cancellation

### Configuration (pkg/config/config.go)
Single `Config` struct with `Default` var. Bound to:
1. YAML config file (`.introspect.yaml`)
2. Environment variables (`INTROSPECTENV_*`)
3. CLI flags (Cobra/Viper)
Precedence: CLI > Env > Config file > Defaults.

### Logging (pkg/logger/logger.go)
Zap-based, implements `logr.Logger` interface. Used throughout via `logger.Log`.
- `config.Default.Development` toggles dev/prod config
- `config.Default.LogLevel` controls verbosity

### Templates & Assets (pkg/assets/)
- Serve templates from filesystem at `/tmpl/` (`assets.ExecuteTemplate`)
- Serve CSS/JS from filesystem at `/css/` (`assets.CSSHandler()`)
- Templates parsed per-request with layout + page
- No build-time embedding or code generation needed

---

## 🧪 Testing Status

**Current State**: Minimal testing
- Only 1 test file: `pkg/operator/useless/controllers/suite_test.go` (Ginkgo + envtest)
- No unit tests for handlers, middleware, config, election, guestbook, mandelbrot, etc.
- `make test` runs `go test ./... -coverprofile cover.out` (passes trivially)
- **No race detector in CI** (`go test -race ./...` not run)

### Critical Gaps
| Package | Risk | Test Status |
|---------|------|-------------|
| election | Race conditions in leader state | None |
| guestbook | Multi-backend switching, config reload | None |
| mandelbrot | CPU-intensive, no benchmarks | None |
| operator | Controller-runtime integration | Ginkgo suite only |
| validate | Admission webhook logic | None |
| middleware | Hijack for WebSockets | None |

---

## 🛠 Development Workflow

### Prerequisites
```bash
go version       # 1.26+
kubectl version
docker --version
kind version
tilt version
make --version
```

### Local Development (Tilt)
```bash
make kind-up      # Create Kind cluster
tilt up           # Start Tilt dev loop
# UI: http://localhost:10350
# App: http://localhost:9090
tilt down         # Cleanup
```

### Build & Test Commands
```bash
make fmt          # go fmt ./...
make vet          # go vet ./...
make test         # go test ./... -coverprofile cover.out
make build        # Docker build (linux/amd64)
make buildx       # Multi-arch docker buildx (linux/amd64,linux/arm64)
make all          # Build binaries (darwin + linux)
make manifests    # Generate CRD/RBAC (controller-gen)
make generate     # Generate deepcopy code
```

### Run Locally (No Cluster)
```bash
go run ./cmd                    # Uses .introspect.yaml config
go run ./cmd --log-level=debug --development
```

---

## 📋 Agent Task Guidelines

### When Adding Features
1. **Create handler in `pkg/<feature>/`** implementing `handler.Handler`
2. **Add template** in `pkg/assets/tmpl/<feature>.html`
3. **Register in `pkg/server/handlers.go`** via `BuildHandlers()`
4. **Add config** to `pkg/config/config.go` if needed
5. **Update `Makefile`** if new code generation needed

### When Fixing Bugs
- **Always run `go test -race ./...`** before committing
- **Check for race conditions** in shared state (election, guestbook, healthz)
- **Verify context propagation** in handlers (pass `ctx` to downstream calls)
- **Validate config** with `viper` bindings and env var mapping

### When Refactoring
- **Preserve `handler.Handler` interface** — central to architecture
- **Keep config centralized** in `pkg/config/`
- **Use `logger.Log`** (logr) not raw Zap
- **Embed templates** via `pkg/assets` — don't read from filesystem at runtime

---

## 🔧 Common Tasks for Agents

### Add a New Feature Endpoint
```bash
# 1. Create pkg/newfeature/newfeature.go
# 2. Implement handler.Handler
# 3. Add template to tmpl/newfeature.html
# 4. Register in pkg/server/handlers.go
# 5. Add config if needed to pkg/config/config.go
# 6. Test: go run ./cmd
```

### Add Unit Tests for a Package
```bash
# 1. Create pkg/<feature>/<feature>_test.go
# 2. Use testify/assert or stdlib testing
# 3. Test handler logic, not HTTP (use httptest)
# 4. Run: go test -race ./pkg/<feature>/...
```

### Fix Race Condition
```bash
# 1. Run: go test -race ./...
# 2. Check shared state (maps, structs accessed from multiple goroutines)
# 3. Add sync.RWMutex or use atomic/sync primitives
# 4. Verify: go test -race -count=10 ./...
```

### Update Dependencies
```bash
go get -u ./...
go mod tidy
go mod verify
# Test: make test && make build
```

---

## ⚠️ Known Issues & Technical Debt

| Area | Issue | Priority |
|------|-------|----------|
| **Testing** | <5% coverage, no unit tests for core packages | 🔴 Critical |
| **Race Detection** | Not in CI; election/guestbook have shared state | 🔴 Critical |
| **Context** | Some handlers use `context.Background()` instead of passed `ctx` | 🟡 High |
| **Config** | Guestbook uses own Viper instance; not unified | 🟡 High |
| **Security** | TLS `InsecureSkipVerify` in kubesec (removed?); no security headers | 🟡 High |
| **Observability** | No distributed tracing; limited metrics | 🟡 High |
| **Templates** | Parsed per-request in some handlers; should be cached | 🟡 Medium |
| **Dependencies** | controller-runtime v0.24.1 (check for updates) | 🟡 Medium |
| **Hot Reload** | No `air`/`reflex` for Tilt Go hot-reload | 🟢 Low |

---

## 🚀 CI/CD Pipeline (`.github/workflows/`)

| Workfile | Purpose |
|----------|---------|
| `build.yml` | Go build, vet, test |
| `release.yml` | Release Please + GoReleaser |
| `codeql.yml` | Static analysis |
| `scorecard.yml` | OpenSSF Scorecard |
| `reuse.yml` | REUSE license compliance |

**Add to CI**: `golangci-lint`, `go test -race`, `govulncheck`, `gosec`

---

## 📝 Code Style Guidelines

- **Go 1.26+**: Use `slog` patterns, `iter.Seq`, `maps/lo` functions where appropriate
- **Error handling**: Wrap with `fmt.Errorf("...: %w", err)`; log with `logger.Log.Error(err, "msg", "key", val)`
- **Context**: Always pass `ctx` through call chains; respect cancellation
- **Naming**: `Handler` for HTTP handlers; `Backend` for storage interfaces; `New()` constructors
- **Interfaces**: Keep small (`Handler`, `Closer`, `Backend`); define in `pkg/handler/`, `pkg/guestbook/`
- **Templates**: Use `assets.ExecuteTemplate(w, "name.html", data)` — reads from filesystem at `tmpl/`

---

## 🔗 Key Files Reference

| File | Purpose |
|------|---------|
| `cmd/introspect/root.go` | CLI config, Viper bindings |
| `cmd/introspect/server.go` | Server command wiring |
| `pkg/server/server.go` | HTTP server lifecycle |
| `pkg/server/handlers.go` | Feature registry |
| `pkg/handler/handler.go` | Handler interface |
| `pkg/config/config.go` | Central config struct |
| `pkg/logger/logger.go` | Logging setup |
| `pkg/assets/assets.go` | Template/CSS filesystem serving |
| `pkg/election/election.go` | Leader election (race-prone) |
| `pkg/guestbook/guestbook.go` | Multi-backend guestbook |
| `pkg/operator/operator.go` | Operator + controller-runtime |
| `Makefile` | Build/test/deploy targets |
| `Tiltfile` | Local dev loop |

---

## 🤖 Agent Instructions Summary

**DO:**
- ✅ Run `go test -race ./...` before completing tasks
- ✅ Follow `handler.Handler` pattern for new endpoints
- ✅ Use `logger.Log` for structured logging
- ✅ Add templates to `pkg/assets/tmpl/`
- ✅ Centralize config in `pkg/config/`

**DON'T:**
- ❌ Create global mutable state without mutex/atomic
- ❌ Use `context.Background()` in request handlers
- ❌ Read templates from filesystem at runtime (use `assets.ExecuteTemplate` which does this efficiently)
- ❌ Add dependencies without `go mod tidy` + verification
- ❌ Skip `make fmt vet` before committing

---

*Generated from codebase analysis. Update as project evolves.*