# Technical Decisions (ADR-lite) — Patient Management System API

Locked choices. The rest of the knowledge base deliberately avoids re-litigating these. If a decision must change, update **this file first**, then the affected docs, and say so in the PR description.

Status legend: **Locked** (do not deviate without a doc update) · **Default** (sensible baseline, cheap to revisit).

---

## D-001 — HIS lookup and local search are two distinct endpoints · Locked

The assignment blends two ideas ("develop APIs to search patient info from HIS" and "implement `/patient/search` scoped to staff hospital"). They are **separate flows**:

| Endpoint                      | Source of truth | Purpose                                                                 |
| ----------------------------- | --------------- | ----------------------------------------------------------------------- |
| `GET /patient/search/:id`     | Hospital A HIS  | Look up one patient by `national_id` or `passport_id`, normalize, **upsert** into local `patients` scoped to the caller's hospital, return the record |
| `POST /patient/search`        | Local Postgres  | Multi-filter search over locally stored patients, **always** filtered by the authenticated staff's hospital |

**Rationale:** HIS is an upstream system of record we do not own; the local DB is the queryable projection. Upsert-on-lookup is what makes the local Patient schema "compatible with hospital data structures" meaningful, and gives `/patient/search` something to search.

**Consequences:**
- The HIS client is called from exactly one service path (`PatientService.LookupFromHIS`).
- `/patient/search` never calls the HIS client — no upstream latency or partial failures on the search path.
- Upsert key: `(hospital, national_id)` when present, else `(hospital, passport_id)`.

## D-002 — Both patient endpoints require authentication · Locked

The requirement only states "Requires login" for `/patient/search`, but `GET /patient/search/:id` writes patient records tagged with a hospital. Without auth there is no hospital to tag, and the endpoint becomes an unauthenticated PII lookup proxy. Auth is required on the whole `/patient` route group.

## D-003 — HTTP methods and shapes · Locked

| Method | Path                    | Auth | Body / params                                    |
| ------ | ----------------------- | ---- | ------------------------------------------------ |
| POST   | `/staff/create`         | ✗    | JSON: `username`, `password`, `hospital`         |
| POST   | `/staff/login`          | ✗    | JSON: `username`, `password`, `hospital`         |
| GET    | `/patient/search/:id`   | ✓    | Path param `id`                                  |
| POST   | `/patient/search`       | ✓    | JSON: eight optional filters                     |

`POST` for the filter search (not `GET` + query string) because the filters carry PII — national IDs, passports, DOB, email. Query strings land in access logs, proxy logs, and browser history; request bodies do not. This is a deliberate trade of REST purity for PII containment, and must be stated in the API Spec deliverable.

**At least one filter is required** on `POST /patient/search`; an empty filter set returns `400 INVALID_INPUT` rather than dumping the hospital's entire patient table.

## D-004 — Response envelope · Locked

Success responses return the resource or collection directly:

```json
{ "data": { ... } }
{ "data": [ ... ], "meta": { "count": 12 } }
```

Errors always use:

```json
{ "error": { "code": "INVALID_INPUT", "message": "human readable, safe to show a user" } }
```

`code` is from the table in [`coding_rules.md`](coding_rules.md). `message` never contains SQL, stack traces, upstream URLs, or PII.

## D-005 — Library choices · Default

| Concern         | Choice                                   | Note                                                        |
| --------------- | ---------------------------------------- | ----------------------------------------------------------- |
| Go              | 1.24 (pin in `go.mod` + Dockerfile)      | Match toolchain in CI                                        |
| HTTP framework  | `github.com/gin-gonic/gin`               | Required by the assignment                                   |
| Postgres driver | `github.com/jackc/pgx/v5` + `pgxpool`    | Do not also import `lib/pq`                                  |
| Migrations      | `golang-migrate/migrate` (plain SQL)     | Files in `migrations/`, `NNNN_name.{up,down}.sql`            |
| JWT             | `github.com/golang-jwt/jwt/v5`           | HS256, secret from config                                    |
| Password hash   | `golang.org/x/crypto/bcrypt`             | Cost from config, default 12                                 |
| Logging         | stdlib `log/slog`                        | JSON handler in prod, text in dev                            |
| Validation      | Gin's bundled `go-playground/validator`  | `binding:"..."` struct tags                                  |
| Test assertions | `github.com/stretchr/testify` (optional) | Hand-written fakes; no mocking framework                     |

Adding a dependency outside this list requires a one-line justification in the PR.

## D-006 — Staff identity and uniqueness · Locked

- A staff member belongs to **exactly one** hospital.
- Uniqueness: `UNIQUE (username, hospital)` — the same username may exist at different hospitals. Login therefore requires all three of `username`, `password`, `hospital`, which matches the requirement's stated input.
- Duplicate create → `409 CONFLICT`.

## D-007 — Token policy · Default

- Access token only; **no refresh token** (out of scope for this assignment — say so in the API Spec).
- TTL 60 minutes, from config (`JWT_TTL`).
- Claims: `sub` (staff id), `hospital`, `iat`, `exp`. Nothing else — no username, no PII.
- Transport: `Authorization: Bearer <token>`.
- **The `hospital` claim is the only authorization input for patient scoping.** A `hospital` field in a request body is never consulted for access control.

## D-008 — Hospital identifier · Locked

`hospital` is a short **code** string (e.g. `hospital-a`), not a display name and not a free-text field. Store it as `TEXT` with a check or a `hospitals` lookup table; either is acceptable, but the same normalized value must be used on Staff, Patient, and the JWT claim. Normalize to lowercase on write.

## D-009 — HIS failure semantics · Locked

| Upstream condition             | Our response              |
| ------------------------------ | ------------------------- |
| 200 + valid body               | `200` with upserted record |
| 404                            | `404 NOT_FOUND`           |
| 4xx (other)                    | `502 BAD_GATEWAY`         |
| 5xx, timeout, decode failure   | `502 BAD_GATEWAY`         |

Upstream status codes and error bodies are logged, never forwarded to the client verbatim.

## D-010 — Time and dates · Locked

- All timestamps stored as `TIMESTAMPTZ` in UTC.
- `date_of_birth` is a calendar date (`DATE`), not a timestamp — no timezone shifting.
- JSON date format: `YYYY-MM-DD` for `date_of_birth`, RFC 3339 for timestamps.

## D-011 — Gender values · Locked

`M` or `F` only, per the HIS contract. Stored as a constrained `TEXT`/enum. Unknown or absent → `NULL`, not an empty string. Do not silently coerce unexpected values; log and store `NULL`.

## D-012 — TLS termination at Nginx uses a self-signed cert · Default

`security_rules.md` §9 states TLS terminates at Nginx. For this assignment there is no real domain, so:

- `make certs` (a dependency of `make setup`) generates a self-signed cert/key into `nginx/certs/` (gitignored, never committed).
- Nginx listens on both `80` and `443` (`NGINX_PORT` / `NGINX_TLS_PORT`, default `8080`/`8443`).
- **No automatic HTTP→HTTPS redirect.** Compose maps both ports to non-standard host ports, so a same-host `301` (`https://$host$request_uri`) would resolve to the wrong port outside the container. In production behind standard `80`/`443` with a CA-issued cert, add that redirect.
- Callers hitting `443` locally will see a self-signed-cert warning (`curl -k` / browser "not secure" click-through) — expected for local assessment, not a bug.

**Consequences:** `nginx/certs/` must exist before `docker compose up`; `make up` and `make setup` cover this. Do not commit the generated `.crt`/`.key`.

---

## Open questions to raise rather than guess

If work requires resolving one of these, ask before implementing:

1. Should a staff member be able to look up a patient in HIS whose record already belongs to a *different* hospital locally? (Current assumption: the upsert is scoped per hospital, so both hospitals may hold their own row.)
2. Is there any patient-facing consent or audit-retention requirement beyond the access log described in [`security_rules.md`](security_rules.md)?
3. Does the reviewer expect a live Hospital A endpoint, or a stubbed/mock upstream for demo? (Current assumption: stubbed, with base URL configurable.)
