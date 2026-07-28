# Security Rules — Patient Management System API

This service handles **patient health identifiers** — national IDs, passport numbers, dates of birth, phone numbers, email. Treat every record as regulated PII. These rules are not optional polish; a reviewer scanning for production readiness will look here first.

Related: [`decisions.md`](decisions.md) · [`coding_rules.md`](coding_rules.md) · [`observability_and_ops.md`](observability_and_ops.md)

---

## 1. The authorization invariant

> **Every patient row returned to a caller must carry the same `hospital` value as the JWT claim of that caller.**

This is the single most important rule in the codebase. It is enforced in the **patient service**, not the handler and not the repository, and it is enforced by *adding a mandatory predicate*, never by filtering after the fact.

```go
// Correct: hospital is a required argument, not an optional filter.
func (s *PatientService) Search(ctx context.Context, hospital string, f SearchFilter) ([]model.Patient, error)
```

Forbidden patterns:

- A repository method that can return patients without a hospital predicate
- Reading `hospital` from the request body or a query param for authorization
- Filtering hospital in Go after fetching a wider result set
- A "superuser" or "all hospitals" bypass flag

Every patient repository query must include `WHERE hospital = $n`. If a query without it is genuinely needed (admin tooling, migrations), it lives in a separate, clearly named method and is not reachable from any HTTP handler.

## 2. Authentication

- Passwords hashed with bcrypt (cost from config, default 12). Never MD5/SHA family, never plaintext, never reversible encryption.
- Password comparison via `bcrypt.CompareHashAndPassword` only — it is constant-time by construction.
- **No user enumeration.** `/staff/login` returns the same `401 UNAUTHORIZED` with the same message and comparable timing for: unknown username, wrong hospital, and wrong password. Run a dummy bcrypt comparison when the staff row is not found so the response time does not reveal existence.
- Minimum password policy at create time: length ≥ 12. Reject common/breached passwords if a list is available; otherwise document the gap.
- Tokens are validated for signature **and** `exp` **and** algorithm — explicitly reject `alg: none` and any algorithm other than HS256.
- No token in URLs, query strings, or logs.

## 3. PII handling

**Never log:** passwords (raw or hashed), full `national_id`, full `passport_id`, full HIS response bodies, JWT strings.

**Safe to log:** `request_id`, `staff_id`, `hospital`, endpoint, status, duration, error class.

When an identifier must appear in a log for debugging, mask it:

```go
func maskID(s string) string  // "1234567890123" -> "*********0123"  (last 4 only)
```

Additional rules:

- HIS responses are decoded into typed structs; never `log.Debug("resp", "body", string(raw))`.
- Error messages returned to clients never echo the searched identifier back.
- Do not add PII fields to metrics labels or trace attributes (unbounded cardinality *and* a leak).
- If a debug/verbose mode dumps payloads, it must be impossible to enable in production config — guard it behind a build tag or a hard check on `ENV != production`.

## 4. Access audit log

Patient data access is auditable. Every successful `/patient/search` and `/patient/search/:id` emits one structured audit record:

```
event=patient_access staff_id=<id> hospital=<code> endpoint=<path>
result_count=<n> filters=<field names only, never values> request_id=<id> ts=<rfc3339>
```

Note `filters` records **which fields were used**, not their values. Audit records go to the same structured log stream; a dedicated table is out of scope for this assignment but should be named as a production follow-up.

## 5. Input handling

- Bind into explicit DTO structs with `binding` tags. Never bind into `map[string]any`.
- Max request body size: 1 MB (`http.MaxBytesReader` or Gin's `MaxMultipartMemory` equivalent) — reject larger with `413`.
- Length-cap every string field at bind time (names ≤ 200, identifiers ≤ 64, email ≤ 320). Unbounded strings become both a DoS vector and a log-flooding vector.
- **Parameterized queries only.** No `fmt.Sprintf` into SQL, ever — including for `ORDER BY`, `LIMIT`, or dynamic `WHERE` clause assembly. Build dynamic filters by appending placeholders and args in lockstep.
- Column and sort names, if ever client-influenced, come from a hardcoded allowlist map.
- Validate `id` in `/patient/search/:id` against an expected charset (alphanumeric, length-bounded) before it reaches the HIS client — do not proxy arbitrary path segments upstream.

## 6. Results and pagination

`POST /patient/search` returns a **bounded** result set. Default limit 50, maximum 200, offset or keyset pagination. An unbounded search over a hospital's patient table is a data-exfiltration primitive as much as a performance problem.

## 7. Rate limiting

- `/staff/login` — per-IP and per-`(username, hospital)` limiting, e.g. 10 attempts / 15 min, with a `429` response. In-memory limiter is acceptable for this assignment; note in the API Spec that production needs a shared store (Redis) because the limit must hold across replicas.
- `/patient/search*` — per-`staff_id` limiting to bound bulk extraction.
- Nginx-level connection and request-rate limits complement, but do not replace, application-level limits.

## 8. Secrets and configuration

- No secrets in the repo. `.env.example` holds keys with placeholder values; `.env` is gitignored.
- `JWT_SECRET` must be ≥ 32 bytes. **Fail fast at startup** if it is missing, short, or equal to a known placeholder value like `changeme` — do not fall back to a default.
- Database credentials come from environment; the Compose file references env vars rather than hardcoding them.
- Any credential that ever appeared in a commit is considered burned and must be rotated, not just removed.

## 9. Transport and headers

- TLS terminates at Nginx. The Go service must not be reachable directly from outside the Compose network — bind it to the internal network only, do not publish its port.
- Nginx sets `X-Forwarded-For` / `X-Forwarded-Proto`; the app trusts proxy headers **only** from the known proxy (`gin.SetTrustedProxies` with the Compose network range, never `nil`, which trusts everything).
- Response headers: `X-Content-Type-Options: nosniff`, `Cache-Control: no-store` on all patient responses, HSTS at Nginx when TLS is on.
- CORS: this is a server-to-server / internal API. Default to **no CORS middleware**. If a browser client is added, use an explicit origin allowlist — never `Access-Control-Allow-Origin: *` alongside credentials.

## 10. Outbound (HIS client)

- Explicit timeout on the HTTP client (connect + total, e.g. 5s total). A `http.Client` with no timeout will hang a request goroutine indefinitely.
- Base URL from config and validated at startup — never assembled from user input.
- Verify TLS certificates. `InsecureSkipVerify` is forbidden, including in dev; use a local CA if a mock upstream needs TLS.
- Cap the response body read (`io.LimitReader`) so a hostile or broken upstream cannot exhaust memory.

## 11. Dependencies and supply chain

- `go mod tidy` clean; `go.sum` committed and never hand-edited.
- Run `govulncheck ./...` before finishing significant changes; treat findings as blocking unless demonstrably unreachable.
- Docker images pinned to a digest or at minimum a specific minor tag — never `:latest`.

## 12. Pre-merge security checklist

- [ ] Every patient query includes a hospital predicate sourced from the JWT claim
- [ ] No client-supplied `hospital` used for authorization anywhere
- [ ] No PII or secrets in log statements added by this change
- [ ] All new SQL is parameterized
- [ ] New string inputs are length-bounded and validated
- [ ] New config secrets validated fail-fast at startup
- [ ] `govulncheck ./...` clean
- [ ] Negative-path tests exist for the new auth/authorization behavior
