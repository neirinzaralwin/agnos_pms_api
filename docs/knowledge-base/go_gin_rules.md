# Go & Gin Rules — Patient Management System API

Authoritative Go language and Gin framework policy. Applies to humans and AI agents on **every** change. Replaces a TypeScript-centric rulebook for this repo.

---

## 1. Toolchain source of truth

Exact library choices are pinned in [`decisions.md`](decisions.md) D-005. Summary:

- **Language:** Go 1.24, pinned identically in `go.mod`, Dockerfile, and CI.
- **Module:** one module at repo root (`go.mod`). Module path should match the GitHub repo path once published.
- **Framework:** [Gin](https://github.com/gin-gonic/gin) for HTTP routing and binding.
- **Database:** PostgreSQL via `jackc/pgx/v5` + `pgxpool`. Do not also import `lib/pq`.
- **Migrations:** `golang-migrate/migrate`, plain SQL.
- **Auth:** `golang-jwt/jwt/v5` (HS256) + `golang.org/x/crypto/bcrypt`.
- **Logging:** stdlib `log/slog`.
- **Container:** multi-stage `Dockerfile`; local stack via Docker Compose (nginx + app + postgres).

Do not introduce a second HTTP framework (Echo, Fiber, chi) without an explicit architecture change and doc update.

---

## 2. Module & package hygiene

- All application code that must stay private lives under `internal/`.
- `cmd/api/main.go` is the composition root: load config, open DB, construct clients/repos/services/handlers, register routes, run server.
- Avoid circular imports; extract shared types to `model` / `dto` / `platform` when needed.
- Run `go mod tidy` after adding or removing dependencies.
- Prefer stdlib when sufficient; justify new dependencies in PR/docs when non-obvious.

---

## 3. Gin conventions

### Router

- Group routes by domain: `/staff`, `/patient`.
- Register auth middleware on the **whole** `/patient` group — both patient endpoints require login ([D-002](decisions.md)).
- Keep route registration near `cmd/api` or a dedicated `internal/handler/router.go` — one obvious place.
- Middleware order is fixed: `Recovery` → `RequestID` → `Logger` → `Timeout` → `Auth` (protected groups only). See [`observability_and_ops.md`](observability_and_ops.md) §7.
- Set `gin.SetTrustedProxies` explicitly to the Compose network range. Never leave it as `nil` (trusts all) — that lets a client spoof `X-Forwarded-For` and defeat per-IP rate limiting.
- `gin.SetMode(gin.ReleaseMode)` when `ENV=production`; debug mode logs routes and request detail that does not belong in production output.

### Context

- Pass `c.Request.Context()` (or derived) into services and repositories — honor cancellation.
- Store auth claims with a private context key type in `middleware` — never use plain string keys for context values.
- Do not pack business logic into Gin middleware beyond auth, logging, and recovery.

### Binding & validation

- Use struct tags (`json`, `form`, `binding`) on request DTOs. Never bind into `map[string]any`.
- Validate required fields for create/login. Patient search fields are all optional individually, but **at least one is required** — reject an empty filter set with `400` ([D-003](decisions.md)).
- Length-cap every string field via `binding:"max=..."`. Unbounded strings are a DoS and log-flooding vector.
- Return `400` with `INVALID_INPUT` on bind/validation failure. The message must not echo submitted PII back to the caller.

### Responses

- Consistent JSON envelopes for success and error — the exact shapes are fixed in [D-004](decisions.md).
- Use explicit status codes: `200`/`201` success, `400` invalid input, `401` unauthenticated, `403` forbidden, `404` not found, `409` conflict, `413` body too large, `429` rate limited, `502` HIS failure, `500` unexpected.
- Emit `[]`, not `null`, for empty collections. Initialize slices before appending.
- Set `Cache-Control: no-store` on every patient response.

### Panic recovery

- Custom recovery middleware, not Gin's default: log the panic value **and stack** at `error` with `request_id`, return the standard envelope with `INTERNAL_ERROR`. Never leak the panic message or stack to the client.

### Server configuration

- Construct `http.Server` explicitly with all timeouts set (see [`observability_and_ops.md`](observability_and_ops.md) §4). `router.Run()` uses an unconfigured server with no timeouts and is **not** acceptable outside throwaway prototypes.

---

## 4. Idiomatic Go

### Errors

- Return `error` as the last result value.
- Wrap with `%w` for inspectability (`errors.Is` / `errors.As`).
- Do not use `panic` for expected control flow (missing row, bad password).

### Concurrency

- Default to request-scoped work on the request goroutine.
- If spawning goroutines, document lifetime and pass context; never leak goroutines per request without coordination.

### Interfaces

- Define small interfaces where the **consumer** needs them (repository, HIS client) to enable tests.
- Do not create interfaces “for every struct” with a single unused implementation — YAGNI.

### Zero values & pointers

- Prefer value types for small DTOs; use pointers when distinguishing missing vs empty is required (e.g. optional search filters).
- Be explicit about nil slices vs empty slices in JSON (`omitempty` policy documented per endpoint).

---

## 5. Persistence

- Migrations are source of truth for schema (`migrations/`).
- Repositories use context-aware queries.
- Transactions: begin in service when multi-step writes need atomicity; pass `Tx` or a unit-of-work abstraction — do not open nested ad-hoc transactions in handlers.
- Map Hospital A field names to DB columns clearly (snake_case in DB is fine).

---

## 6. Hospital A client

- Implement behind an interface, e.g. `SearchByID(ctx, id string) (*HISPatient, error)`.
- Configure base URL and timeouts via `config`.
- Treat non-2xx and decode failures as wrapable errors → service maps to `BAD_GATEWAY` or domain-specific handling.
- Never log full PII payloads at info level.

---

## 7. Auth (Gin + JWT or equivalent)

- Login issues a signed token (or session) embedding `staff_id` and `hospital`.
- Middleware validates token and sets claims on context.
- Password verify with constant-time compare via bcrypt (or argon2); cost factor from config with a secure default.
- `/staff/create` and `/staff/login` remain public; `/patient/search` requires auth.

---

## 8. Testing rules

- File name: `*_test.go` in the same package (or `package foo_test` for black-box).
- Table-driven cases with `t.Run` names describing behavior.
- Cover each API: happy path + failures (bad credentials, unauthorized search, validation errors, not found, HIS error if applicable).
- Prefer fakes/stubs implementing interfaces over heavy frameworks.
- Race detector for concurrent code when relevant: `go test -race ./...`.

---

## 9. Static analysis checklist (before finish)

```bash
gofmt -w .
go vet ./...
go test ./... -race -cover
govulncheck ./...
# when configured:
golangci-lint run
```

Full testing requirements — coverage targets and the required case list per endpoint — are in [`testing_strategy.md`](testing_strategy.md).

---

## 10. Forbidden

- Import cycles between `handler`, `service`, and `repository`
- Using Gin’s `Context` inside repository or HIS client packages
- Global mutable state for DB pools or clients without clear init in `main`
- Ignoring `error` return values (`_ = do()`) without a comment justifying why
- `SELECT *` without need; prefer explicit columns as schema grows
- Committing `vendor/` unless the team explicitly chooses vendoring
- `router.Run()` in place of a configured `http.Server` with timeouts
- `gin.SetTrustedProxies(nil)` or leaving trusted proxies unset
- Building SQL with `fmt.Sprintf`, including for dynamic filter assembly
- `http.Client` without a timeout, or `InsecureSkipVerify: true` anywhere

See also: [coding_rules.md](coding_rules.md), [dependency_map.md](dependency_map.md), [security_rules.md](security_rules.md), [observability_and_ops.md](observability_and_ops.md).
