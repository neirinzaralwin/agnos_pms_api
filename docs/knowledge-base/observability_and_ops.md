# Observability & Operations — Patient Management System API

What separates a working assignment from a production-grade one: the service must be observable, shut down cleanly, refuse to start when misconfigured, and survive its dependencies being slow or absent.

Related: [`decisions.md`](decisions.md) · [`security_rules.md`](security_rules.md) · [`go_gin_rules.md`](go_gin_rules.md)

---

## 1. Health endpoints

Two endpoints, different meanings. Both are **unauthenticated** and both are excluded from access logging and rate limits.

| Path       | Checks                                     | Used by                          |
| ---------- | ------------------------------------------ | -------------------------------- |
| `/healthz` | Process is alive. No dependency checks.    | Container liveness, Nginx upstream |
| `/readyz`  | DB pool reachable (`Ping` with 2s timeout) | Compose `depends_on: healthy`, deploys |

`/readyz` returns `503` with a body naming the failing dependency (dependency name only — no DSN, no credentials). Never make `/healthz` depend on Postgres: a brief DB blip should not cause the orchestrator to kill an otherwise healthy process.

Optionally expose `/version` returning build metadata injected via `-ldflags` (commit SHA, build time). Useful and free.

## 2. Startup contract

`cmd/api/main.go` in order:

1. Load and **validate** config — every required var present, `JWT_SECRET` length checked, base URLs parseable. Missing or invalid → log one clear error and `os.Exit(1)`. Never boot in a degraded state with defaults for secrets.
2. Construct the logger before anything that might log.
3. Open the DB pool and `Ping` with a bounded timeout. Fail fast if unreachable at startup.
4. Construct clients → repositories → services → handlers → router (composition root; no globals).
5. Start the HTTP server.
6. Log exactly one startup line: version, port, env, log level. Never log the DSN or secrets.

## 3. Graceful shutdown

Non-negotiable for a service that writes to a database.

```go
srv := &http.Server{Addr: addr, Handler: router, /* timeouts below */}

go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Error("server error", "error", err)
        os.Exit(1)
    }
}()

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
<-ctx.Done()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Error("graceful shutdown failed", "error", err)
}
pool.Close()
```

In-flight requests finish; new connections are refused; the pool closes after the server, not before.

## 4. Timeouts — every boundary, no exceptions

An untimed boundary is an outage waiting for a slow dependency.

| Boundary            | Setting                                   | Baseline |
| ------------------- | ----------------------------------------- | -------- |
| HTTP server read    | `ReadHeaderTimeout` / `ReadTimeout`       | 5s / 15s |
| HTTP server write   | `WriteTimeout`                            | 20s      |
| HTTP server idle    | `IdleTimeout`                             | 60s      |
| Per-request ceiling | context timeout middleware                | 10s      |
| DB query            | `context.WithTimeout` in repository calls | 3s       |
| HIS outbound        | `http.Client.Timeout`                     | 5s       |
| Shutdown drain      | `srv.Shutdown` context                    | 15s      |

`ReadHeaderTimeout` in particular must be set — its absence is a Slowloris exposure and a common lint finding (`gosec` G112).

Every repository and client method takes `context.Context` as its first parameter and passes it through. A method that ignores the context it was given is a bug.

## 5. Connection pool

Configure `pgxpool` explicitly rather than accepting defaults:

- `MaxConns` — start at 10 per replica; it must be sized so `replicas × MaxConns` stays under Postgres `max_connections` with headroom.
- `MinConns` 2, `MaxConnLifetime` 30m, `MaxConnIdleTime` 5m, `HealthCheckPeriod` 1m.
- All from config with documented defaults.

## 6. Structured logging

`log/slog` with a JSON handler in production, text handler in dev, level from `LOG_LEVEL` (default `info`).

Standard fields on request-scoped logs: `request_id`, `method`, `path`, `status`, `duration_ms`, and where authenticated, `staff_id` and `hospital`.

| Level  | Use                                                                     |
| ------ | ----------------------------------------------------------------------- |
| `debug`| Local development detail. Off in production.                            |
| `info` | Request completion, startup, shutdown, audit events.                     |
| `warn` | Handled degradation — upstream retry, rate limit hit, validation anomaly. |
| `error`| Unexpected failures, 5xx responses, panics.                              |

A client's bad input is a `warn` at most, never `error` — 4xx responses logged at `error` level make error rates meaningless as an alerting signal.

Carry a `*slog.Logger` on the request context so downstream layers inherit `request_id` automatically instead of threading it manually.

## 7. Request correlation

Middleware, ordered: `Recovery` → `RequestID` → `Logger` → `Timeout` → `Auth` (on protected groups).

`RequestID` accepts an inbound `X-Request-ID` when present (so Nginx or a caller can correlate), otherwise generates a UUID. Always echo it back in the response header. It appears in every log line for that request and in `500` response bodies so a user-reported error can be traced to a log entry.

## 8. Panic recovery

Custom recovery middleware — not just Gin's default. It must log the panic value **and stack** at `error` with the `request_id`, then return the standard error envelope with `INTERNAL_ERROR` and a `500`. It must never leak the panic message or stack to the client.

## 9. Metrics — scope note

Full Prometheus instrumentation is beyond this assignment. Structure the code so it can be added without refactoring: keep the middleware chain as the single place where request duration and status are observed. State this explicitly in the planning deliverable — naming a deliberate scope boundary reads as engineering judgment; silently omitting it reads as an oversight.

If adding metrics anyway: request count/duration by route and status, DB pool saturation, HIS call duration and error rate. Never put PII or unbounded values in label sets.

## 10. Docker

**Dockerfile — multi-stage, non-root, minimal:**

- Build stage on `golang:1.24-alpine` (or matching Debian); final stage on `gcr.io/distroless/static` or `alpine`.
- `CGO_ENABLED=0` for a static binary.
- Copy `go.mod`/`go.sum` and `go mod download` **before** copying source — keeps the dependency layer cached.
- `-ldflags="-s -w -X main.version=$VERSION"` to strip and stamp.
- Run as a non-root user (`USER 65532:65532` on distroless, or a created user on alpine).
- `.dockerignore` covering `.git`, `docs`, `*.md`, `.env`, test artifacts.
- Never bake secrets into the image or into build args that persist in layer history.

**docker-compose.yml — nginx + app + postgres:**

- Postgres: named volume for data, healthcheck via `pg_isready`, credentials from env.
- App: `depends_on: postgres: condition: service_healthy`, healthcheck hitting `/healthz`, restart policy `unless-stopped`, **no published host port** — reachable only through Nginx on the internal network.
- Nginx: publishes 80/443, proxies to the app service by Compose DNS name, sets `X-Request-ID` and forwarding headers, applies `client_max_body_size 1m` and a basic rate limit zone.
- No hardcoded credentials in the Compose file; reference `${VAR}` with an `.env.example` committed.

## 11. Migrations

- Versioned SQL under `migrations/`, both `.up.sql` and `.down.sql`. A migration without a tested down is incomplete.
- Applied as an explicit step — a dedicated Compose one-shot service or documented `make migrate` — **not** silently on application startup, so a rollout can't half-migrate under multiple replicas racing.
- Migrations are append-only. Never edit an applied migration; write a new one.
- Every index needed by a documented search filter is created in a migration, not assumed.

## 12. Ops checklist before finishing

- [ ] `/healthz` and `/readyz` present, `/readyz` checks the DB, `/healthz` does not
- [ ] Config validation fails fast with a clear message on missing/short secrets
- [ ] Graceful shutdown wired to SIGTERM/SIGINT with a bounded drain
- [ ] All six server/client timeouts set, including `ReadHeaderTimeout`
- [ ] Every repository and client method accepts and honors `context.Context`
- [ ] Request ID generated or propagated, echoed in the response header, present in logs
- [ ] Custom recovery middleware logs stack, returns generic `500`
- [ ] Dockerfile multi-stage, non-root, no `:latest`, `.dockerignore` present
- [ ] Compose: app port not published, Postgres healthcheck, no hardcoded credentials
- [ ] Migrations have down files and are applied as an explicit step
