# 2. API Spec

Base URL (local Compose): `http://localhost:8080` or `https://localhost:8443` (self-signed).

All JSON. Auth for patient routes: `Authorization: Bearer <access_token>`.

---

## Response envelopes

**Success (single resource):**

```json
{ "data": { } }
```

**Success (collection):**

```json
{
  "data": [ ],
  "meta": { "count": 12 }
}
```

**Error:**

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "human readable, safe to show a user"
  }
}
```

`message` never includes SQL, stack traces, upstream URLs, or PII.

### Error codes

| Code | HTTP | Use |
| ---- | ---- | --- |
| `INVALID_INPUT` | 400 | Validation / bind failures; empty search filters |
| `UNAUTHORIZED` | 401 | Missing/invalid credentials or token |
| `FORBIDDEN` | 403 | Authenticated but not allowed |
| `NOT_FOUND` | 404 | Missing resource; HIS 404 |
| `CONFLICT` | 409 | Duplicate staff `(username, hospital)` |
| `PAYLOAD_TOO_LARGE` | 413 | Body over 1 MB |
| `RATE_LIMITED` | 429 | Edge or app rate limit |
| `INTERNAL_ERROR` | 500 | Unexpected |
| `BAD_GATEWAY` | 502 | HIS upstream failure / timeout / decode error |

---

## Endpoints

### `POST /staff/create` — public

Create a staff member bound to one hospital.

**Request**

```json
{
  "username": "alice",
  "password": "at-least-12-chars",
  "hospital": "hospital-a"
}
```

- `hospital` is a short code (normalized lowercase), not a display name.
- Password minimum length ≥ 12.
- Uniqueness: `(username, hospital)`.

**Responses:** `201` with created staff (no password hash) · `400 INVALID_INPUT` · `409 CONFLICT`

---

### `POST /staff/login` — public

**Request**

```json
{
  "username": "alice",
  "password": "at-least-12-chars",
  "hospital": "hospital-a"
}
```

**Response `200`**

```json
{
  "data": {
    "access_token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

**JWT claims (HS256):** `sub` (staff id), `hospital`, `iat`, `exp`. No refresh token (out of scope).

Failed login (unknown user, wrong hospital, wrong password) returns the same `401 UNAUTHORIZED` with comparable timing.

---

### `GET /patient/search/:id` — auth required

HIS lookup by `national_id` or `passport_id`, normalize, **upsert** into local `patients` tagged with the caller's JWT `hospital`, return the record.

**Path:** `id` — alphanumeric, length-bounded.

**Responses:** `200` patient · `401` · `404 NOT_FOUND` (HIS 404) · `502 BAD_GATEWAY` (HIS 4xx/5xx other, timeout, decode failure)

Upstream status bodies are logged, never forwarded verbatim.

---

### `POST /patient/search` — auth required

Local Postgres search only — **never** calls HIS. Always scoped to JWT `hospital`.

**Request** (all fields optional; **at least one required**)

```json
{
  "national_id": "",
  "passport_id": "",
  "first_name": "",
  "middle_name": "",
  "last_name": "",
  "date_of_birth": "1990-01-15",
  "phone_number": "",
  "email": "",
  "limit": 50,
  "offset": 0
}
```

Why `POST` instead of `GET` + query string: filters carry PII; query strings appear in access logs, proxies, and browser history.

Optional pagination: `limit` (default 50, max 200) and `offset` (default 0). A `hospital` field in the body, if present, is **ignored** for authorization.

**Responses:** `200` `{ "data": [ ... ], "meta": { "count": N } }` · `400` if no filters · `401`

Result cap: default 50, max 200.

---

### `GET /healthz` — public

Liveness. No dependency checks. Not rate-limited / not access-logged at the edge probe path.

### `GET /readyz` — public

Readiness. Pings Postgres; `503` with `{ "dependency": "postgres" }` if down (no DSN/credentials in body).

---

## Patient flows (distinct)

```text
GET  /patient/search/:id  →  HIS  →  upsert local (caller's hospital)  →  return
POST /patient/search      →  local DB only (WHERE hospital = jwt.hospital)
```

---

## Auth & scoping rules

- Transport: `Authorization: Bearer <token>`.
- Entire `/patient` group requires auth (HIS lookup writes hospital-tagged rows).
- **Hospital for authorization comes only from the JWT claim**, never from a request field.
- Patient responses set `Cache-Control: no-store`.

---

## External: Hospital A HIS

- `GET https://hospital-a.api.co.th/patient/search/{id}`
- `{id}`: national or passport id
- Fields: Thai/English names, `date_of_birth`, `patient_hn`, `national_id`, `passport_id`, `phone_number`, `email`, `gender` (`M` \| `F`)

Base URL and timeout are configurable (`HOSPITAL_A_BASE_URL`, `HOSPITAL_A_TIMEOUT`).

In local Compose, `HOSPITAL_A_BASE_URL` is overridden to `http://hospital-a-mock:9090` (see `cmd/mockhis`). Fixture ids: `1100700123456`, `A1234567`, `ODDGENDER`. Reserved: `500ERROR` → 500, `BADJSON` → undecodable body.

---

## Rate limiting

| Layer | Scope | Baseline |
| ----- | ----- | -------- |
| Nginx | Per-IP edge (`limit_req`) | 10 r/s burst 20 |
| App | Per-IP on `/staff/login` | 10 / 15 min |
| App | Per-`(username, hospital)` on login | 10 / 15 min |
| App | Per-`staff_id` on `/patient/*` | 60 / min |

App limiters are in-memory (acceptable for this assignment). Production across replicas needs a shared store (e.g. Redis) — noted here as a deliberate scope boundary.
