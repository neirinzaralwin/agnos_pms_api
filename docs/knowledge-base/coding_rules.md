# Coding Rules — Patient Management System API

## Language & Style

- **Go** (see [`go_gin_rules.md`](go_gin_rules.md) for version, modules, Gin specifics).
- Format with `gofmt` / `goimports`. Match surrounding file style.
- Lint with `go vet ./...` and `golangci-lint run` when configured. Fix the cause; do not blanket-disable linters.
- No `fmt.Println`, `log.Print*`, or unstructured prints in committed application code under `cmd/` or `internal/`. Use the structured logger from `internal/platform`.

## Naming Conventions

### Packages

- Short, lowercase, no underscores: `handler`, `service`, `repository`, `hospitala`.
- Avoid stuttering: `hospitala.Client` not `hospitala.HospitalAClient` when the package already names the system.

### Files

| Element    | Pattern                 | Example                 |
| ---------- | ----------------------- | ----------------------- |
| Handler    | `<domain>_handler.go`   | `staff_handler.go`      |
| Service    | `<domain>_service.go`   | `patient_service.go`    |
| Repository | `<domain>_repository.go`| `patient_repository.go` |
| Model      | `<entity>.go`           | `staff.go`              |
| Test       | `<file>_test.go`        | `staff_handler_test.go` |

### Symbols

- Exported: `PascalCase`. Unexported: `camelCase`.
- Interfaces: often `Reader` / `Writer` / role names (`PatientRepository`, `HospitalAClient`) — noun describing behavior.
- Constants: `PascalCase` for exported; `camelCase` or `PascalCase` for unexported; `SCREAMING_SNAKE` only for env-like or protocol constants when conventional.

## Handlers (Gin)

- Thin: bind → validate → call service → write JSON / status.
- Do not put SQL, password hashing policy details beyond calling service, or HIS HTTP here.
- Use `ShouldBindJSON` / `ShouldBindQuery` (or project-chosen binder) and return `400` on bind failure.
- Map domain errors to HTTP in one place (handler helpers or middleware) — stable JSON shape `{ "code", "message" }` (plus status).

## Services

- Orchestration and business rules only.
- Inject dependencies via constructor (`NewPatientService(repo, his, log)`).
- Enforce **same-hospital** patient visibility using auth context hospital.
- Hash passwords with a proven library (e.g. bcrypt); never store plaintext.
- Keep methods focused; extract helpers when a function grows large (~50+ lines as a smell, not a hard law).

## Repositories

- Encapsulate all SQL for the owned table(s).
- Accept `context.Context` on every DB method.
- Return domain models or explicit DTOs — do not leak driver-specific types to handlers.
- Use parameterized queries only — never string-concatenate user input into SQL.

## Errors

- Wrap with context: `fmt.Errorf("create staff: %w", err)`.
- Define sentinel or typed errors for expected cases (`ErrNotFound`, `ErrUnauthorized`, `ErrConflict`, `ErrInvalidInput`).
- Handlers translate known errors → HTTP; unknown errors → `500` with generic client message; log full error server-side.
- Do not swallow errors silently. If continuing after failure, log at warn/error with reason.

### Suggested error codes (API contract)

| Code                 | Typical status | Use                            |
| -------------------- | -------------- | ------------------------------ |
| `INVALID_INPUT`      | 400            | Validation / bind failures      |
| `UNAUTHORIZED`       | 401            | Missing/invalid credentials     |
| `FORBIDDEN`          | 403            | Authenticated but not allowed   |
| `NOT_FOUND`          | 404            | Missing resource                |
| `CONFLICT`           | 409            | Duplicate staff, etc.           |
| `PAYLOAD_TOO_LARGE`  | 413            | Request body over the 1 MB cap  |
| `RATE_LIMITED`       | 429            | Login or search rate limit hit  |
| `INTERNAL_ERROR`     | 500            | Unexpected                      |
| `BAD_GATEWAY`        | 502            | HIS upstream failure            |

Envelope shape is fixed in [`decisions.md`](decisions.md) D-004. HIS status → our status mapping is in D-009.

## Logging

- Structured fields: `request_id`, `staff_id`, `hospital`, `error`. **Never** log raw passwords, tokens, or full national/passport IDs — mask to last 4 when an identifier is needed for debugging.
- JSON logs in production; human-friendly in local dev via config.
- Env: `LOG_LEVEL` (default `info`).
- A client's bad input is `warn` at most, never `error` — 4xx logged as `error` makes error-rate alerting useless.
- Full PII and level rules: [`security_rules.md`](security_rules.md) §3. Field conventions and middleware ordering: [`observability_and_ops.md`](observability_and_ops.md) §6–7.

## Configuration

- Load once in `internal/config` at startup; validate required vars fail-fast.
- Pass config structs into constructors — avoid repeated `os.Getenv` in hot paths.

## Testing

- Unit tests for each API behavior: **positive and negative** cases (requirement, and an explicit evaluation criterion).
- Table-driven tests preferred.
- Fake at repository and HIS client interfaces — never hit real Hospital A; live Postgres only behind the `integration` build tag.
- Run `go test ./... -race -cover` before finishing changes.

Coverage thresholds and the required case list for every endpoint are in [`testing_strategy.md`](testing_strategy.md) — treat that document as the checklist, not this section.

## Forbidden

- Handler → repository or HIS client imports
- Plaintext passwords in DB or logs
- Client-supplied hospital overriding auth hospital for authorization
- Committing secrets, `.env` with real credentials, or private keys
- Editing generated artifacts or `go.sum` by hand without `go mod tidy`
- Disabling vet/lint globally to hide issues
