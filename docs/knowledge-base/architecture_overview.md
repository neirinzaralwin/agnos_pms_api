# Architecture Overview — Patient Management System API

This document defines the canonical architecture for the **Hospital Middleware** API (Agnos candidate assignment). AI agents must respect this architecture when generating, refactoring, or extending code.

**Product requirements:** [Project_Requirement.md](../../Project_Requirement.md)

## System Purpose

Backend middleware that lets hospital **staff** authenticate and **search patients** scoped to their own hospital. Patient data is stored locally (Postgres) in a schema compatible with Hospital Information System (HIS) payloads, and is populated by lookups against the external **Hospital A** HIS API.

> **Pinned decisions live in [`decisions.md`](decisions.md).** Where this document says "the flow is X", that flow is locked there with its rationale. Do not re-decide it per PR.

## Runtime Topology

```
Staff client ──HTTP──► Nginx ──proxy──► Gin API (Go) ──► PostgreSQL
                                              │
                                              └──► Hospital A HIS API
                                                   (GET /patient/search/{id})
```

| Component   | Role                                                          |
| ----------- | ------------------------------------------------------------- |
| Nginx       | Reverse proxy, TLS termination (when configured), static gate |
| Gin API     | Auth, staff CRUD-lite, patient search, HIS client             |
| PostgreSQL  | Staff and Patient persistence                                 |
| Hospital A  | External HIS; lookup by `national_id` or `passport_id`        |

Compose all three local services via `docker-compose.yml` (nginx + golang + postgres).

## Domain Modules

| Domain / package              | Responsibility                                                                 |
| ----------------------------- | ------------------------------------------------------------------------------ |
| `staff`                       | Create staff with credentials; login; hospital binding                         |
| `patient`                     | Search patients; enforce same-hospital scope as authenticated staff            |
| `client/hospitala`            | HTTP adapter for Hospital A HIS (`GET .../patient/search/{id}`)                |
| `middleware` / auth           | JWT (or equivalent) login required for patient search; inject staff + hospital |
| `config` / `platform`         | Env config, DB pool, logger, shared error types                                |

## Layering

```
handler (Gin)  →  service  →  repository  →  PostgreSQL
                     │
                     └──► client/hospitala  →  Hospital A API
```

- **Handlers:** bind/validate request, call service, map to HTTP status + JSON. No SQL, no HIS HTTP.
- **Services:** business rules (password hashing, JWT issuance, hospital-scoped search, optional HIS enrichment).
- **Repositories:** SQL only for owned tables; return domain models / DTOs.
- **HIS client:** sole place that calls `https://hospital-a.api.co.th/...` (base URL from config).

## Deliverable HTTP APIs

| Method | Path                   | Auth     | Input                              | Notes                                                    |
| ------ | ---------------------- | -------- | ---------------------------------- | -------------------------------------------------------- |
| POST   | `/staff/create`        | Public   | `username`, `password`, `hospital` | Create staff for a hospital                              |
| POST   | `/staff/login`         | Public   | `username`, `password`, `hospital` | Returns a JWT carrying `staff_id` + `hospital`           |
| GET    | `/patient/search/:id`  | Required | Path param `id`                    | HIS lookup → normalize → upsert locally → return record  |
| POST   | `/patient/search`      | Required | Optional filters (see below)       | Local DB only; always scoped to staff's hospital         |
| GET    | `/healthz`             | Public   | —                                  | Liveness; no dependency checks                           |
| GET    | `/readyz`              | Public   | —                                  | Readiness; pings DB                                      |

Method and shape rationale: [D-003](decisions.md). `POST` is used for the filter search because the filters carry PII that must not land in access logs or browser history.

### The two patient flows are distinct

```
GET  /patient/search/:id  →  service  →  hospitala client  →  Hospital A
                                     └─►  repository (upsert, tagged with caller's hospital)

POST /patient/search      →  service  →  repository  →  Postgres   (never touches HIS)
```

Keeping HIS off the filter-search path means upstream latency and outages cannot degrade the primary search endpoint.

### Patient search filters (all optional, at least one required)

`national_id`, `passport_id`, `first_name`, `middle_name`, `last_name`, `date_of_birth`, `phone_number`, `email`

An empty filter set is rejected with `400 INVALID_INPUT` — it would otherwise export the hospital's entire patient table. Results are capped (default 50, max 200).

### Hospital A HIS contract (external)

- Route: `GET https://hospital-a.api.co.th/patient/search/{id}`
- `{id}`: `national_id` or `passport_id`
- Response fields: Thai/English names, `date_of_birth`, `patient_hn`, `national_id`, `passport_id`, `phone_number`, `email`, `gender` (`M` \| `F`)

Local **Patient** schema must be compatible with these fields plus a **hospital** association so staff scoping works.

## Auth Model

- Staff belong to exactly one **hospital** (normalized lowercase code stored on Staff — [D-008](decisions.md)).
- Login verifies `username` + `password` + `hospital`; failures are indistinguishable from one another ([`security_rules.md`](security_rules.md) §2).
- Protected routes require a valid JWT carrying `sub` (staff id) and `hospital`, nothing more.
- Patient queries **must** filter by the authenticated staff’s hospital — never accept a client-supplied hospital override for authorization. This invariant is stated in full in [`security_rules.md`](security_rules.md) §1 and must have a dedicated regression test ([`testing_strategy.md`](testing_strategy.md) §3).

## Data Model (logical)

### Staff

- `id`, `username`, `password_hash` (bcrypt), `hospital`, `created_at`, `updated_at`
- `UNIQUE (username, hospital)` — the same username may exist at different hospitals ([D-006](decisions.md))
- `hospital` scopes patient visibility

### Patient

- Fields aligned with the Hospital A response: `first_name_th`, `middle_name_th`, `last_name_th`, `first_name_en`, `middle_name_en`, `last_name_en`, `date_of_birth` (`DATE`), `patient_hn`, `national_id`, `passport_id`, `phone_number`, `email`, `gender` (`M`/`F`/`NULL`)
- `hospital` — **required**, non-null; the basis of all scoping
- Upsert key: `(hospital, national_id)` when present, else `(hospital, passport_id)` — enforced with partial unique indexes
- Indexes on `(hospital, national_id)`, `(hospital, passport_id)`, `(hospital, last_name_en)`, `(hospital, date_of_birth)` — every documented filter that will be used at scale gets an index in a migration
- `created_at`, `updated_at` (`TIMESTAMPTZ`, UTC)

## Cross-Cutting Concerns

- **Config:** env-based (`DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `BCRYPT_COST`, `HOSPITAL_A_BASE_URL`, `HOSPITAL_A_TIMEOUT`, `PORT`, `LOG_LEVEL`, `ENV`) via `internal/config`, validated fail-fast at startup — no scattered `os.Getenv` in handlers/services.
- **Logging:** `log/slog` structured logger in `internal/platform` — no `fmt.Println` in committed app code. PII rules in [`security_rules.md`](security_rules.md) §3.
- **Errors:** typed/sentinel or wrapped domain errors mapped to HTTP in one place; stable envelope per [D-004](decisions.md).
- **Operability:** health/readiness endpoints, graceful shutdown, and timeouts at every boundary — see [`observability_and_ops.md`](observability_and_ops.md).
- **Tests:** unit tests for each API path, positive and negative — required cases enumerated in [`testing_strategy.md`](testing_strategy.md).

## Prefer

- One binary under `cmd/api`
- Interfaces at service boundaries for repositories and HIS client (test doubles)
- Migrations as versioned SQL under `migrations/`, applied as an explicit step

## Avoid

- Business logic in Gin handlers
- Calling HIS from handlers, repositories, or the `POST /patient/search` path
- Cross-hospital patient leakage
- New top-level packages without updating this doc and `folder_structure.md`

## Deliverables Mapping

The assignment asks for three planning documents. This knowledge base is the source material for them:

| Deliverable        | Source                                                                                   |
| ------------------ | ---------------------------------------------------------------------------------------- |
| Project Structure  | [`folder_structure.md`](folder_structure.md) + [`module_boundaries.md`](module_boundaries.md) |
| API Spec           | This document's endpoint table + [`decisions.md`](decisions.md) D-003/D-004/D-009 + error codes in [`coding_rules.md`](coding_rules.md) |
| ER Diagram         | The Data Model section above                                                             |

Keep the exported/shared copies generated from these files, not written independently — divergence between the repo docs and the shared docs is exactly what the "documentation clarity" criterion penalizes.
