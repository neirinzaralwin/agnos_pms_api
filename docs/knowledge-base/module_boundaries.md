# Module Boundaries — Patient Management System API

## Process Level

### Public surface

- HTTP API only (via Gin, behind Nginx — the Go service is never published directly)
- Documented routes: `POST /staff/create`, `POST /staff/login`, `GET /patient/search/:id`, `POST /patient/search`, `GET /healthz`, `GET /readyz`
- Both `/patient` routes require authentication ([`decisions.md`](decisions.md) D-002); health endpoints are public and unlogged

### Private surface

- Everything under `internal/` — other Go modules must not import it
- SQL schema details, password hashes, JWT secrets, HIS credentials

## Package Boundaries

### `internal/handler`

- **Public (within app):** Gin handler constructors and route registration helpers
- **Private:** binding details, status mapping helpers
- May depend on: `service`, `dto`/`model`, `middleware` (via engine), `platform` errors
- Must **not** import: `repository`, `client/hospitala`, database drivers

### `internal/service`

- **Public:** service structs/interfaces used by handlers
- Orchestrates repositories + HIS client + auth rules (hospital scope)
- May depend on: `repository`, `client/hospitala`, `model`, `platform`, `config`
- Must **not** import: `handler`, Gin types (prefer plain Go types at the boundary)

### `internal/repository`

- **Public:** repository interfaces/implementations for one aggregate
- Owns writes/reads for Staff or Patient tables
- May depend on: `model`, `platform` (DB), stdlib/`database/sql` (or chosen driver)
- Must **not** import: `handler`, `service`, `client/hospitala`

### `internal/client/hospitala`

- **Public:** `Client` interface + constructor
- Sole owner of Hospital A HTTP I/O and response decoding
- May depend on: `config`, stdlib/`net/http`, small DTO types for HIS payload
- Must **not** import: `handler`, `repository`, Gin

### `internal/middleware`

- Auth, correlation IDs, recovery
- Sets authenticated staff identity + **hospital** on request context
- Must not contain patient search business logic

### `internal/config` / `internal/platform`

- Shared infrastructure only
- Must **not** import domain handlers/services/repositories

## Data ownership

| Table / aggregate | Writer repository | Readers                                      |
| ----------------- | ----------------- | -------------------------------------------- |
| Staff             | Staff repository  | Staff service (login/create); auth middleware via service/repo as designed |
| Patient           | Patient repository| Patient service only                         |

- Single writer per table.
- Other packages read through the owning service or an exported repository interface — do not duplicate SQL.

## Authorization boundary

- Hospital scope is enforced in the **patient service** using the hospital from the **authenticated context**, never from a request field.
- Enforcement is by *adding a required predicate*, not by post-filtering a wider result set.
- Every patient repository method takes `hospital` as a mandatory parameter — there is no repository method capable of returning patients across hospitals reachable from an HTTP handler.
- HIS lookups do not bypass hospital scoping: an upserted record is tagged with the **caller's** hospital.

Full statement of the invariant: [`security_rules.md`](security_rules.md) §1.

## Checklist before merge

- [ ] Handlers do not import repositories or HIS client
- [ ] Repositories do not import services or handlers
- [ ] Only `client/hospitala` performs Hospital A HTTP calls
- [ ] `POST /patient/search` does not touch the HIS client at all
- [ ] Every patient query filters by staff hospital from the JWT claim
- [ ] No repository method returns patients without a hospital predicate
- [ ] No new global middleware registered outside the composition root (`cmd/api`)
