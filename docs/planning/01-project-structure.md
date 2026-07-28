# 1. Project Structure

Hospital Middleware API — Go module layout, layering, and package responsibilities.

## Runtime topology

```text
Staff client ──HTTP(S)──► Nginx ──proxy──► Gin API (Go) ──► PostgreSQL
                                                 │
                                                 └──► Hospital A HIS (Compose: hospital-a-mock)
                                                      GET /patient/search/{id}
```

Compose runs **nginx + api + postgres + migrate (one-shot) + hospital-a-mock**. The API and mock HIS are not published on host ports; clients reach the API only through Nginx.

## Repository layout

```text
.
├── cmd/
│   ├── api/main.go              # Composition root: config → DB → HIS client → services → handlers → router → http.Server
│   └── mockhis/main.go          # Local Hospital A stub for Compose demos
├── internal/
│   ├── staff/
│   │   ├── domain/               # Staff aggregate, Username/Password VOs, Repository port
│   │   ├── application/          # Service: Create, Login
│   │   ├── infrastructure/postgres/  # Repository implementation
│   │   └── transport/http/       # Handler, DTOs
│   ├── patient/
│   │   ├── domain/               # Patient aggregate, Gender/LookupID/SearchCriteria VOs, Repository + HISClient ports
│   │   ├── application/          # Service: LookupFromHIS, Search
│   │   ├── infrastructure/
│   │   │   ├── postgres/          # Repository implementation
│   │   │   └── hospitala/         # Hospital A HTTP adapter (anti-corruption layer)
│   │   └── transport/http/       # Handler, DTOs
│   ├── shared/                   # Shared kernel: apperr, hospital (Code VO), httpkit (response envelope)
│   ├── httpapi/                  # Composition root for HTTP: router, health
│   ├── middleware/                # Recovery, request ID, logger, timeout, JWT auth, rate limit
│   ├── config/                    # Env load + fail-fast validation
│   └── platform/                  # DB pool, slog logger, JWT, infra sentinel errors
├── migrations/                    # Versioned SQL (.up.sql / .down.sql)
├── nginx/                         # Reverse-proxy config + TLS certs (generated)
├── docs/planning/                 # This planning set (interviewer-facing)
├── docker-compose.yml             # nginx + api + postgres + migrate + hospital-a-mock
├── Dockerfile                     # Multi-stage: targets `api` and `mockhis`
├── Makefile
├── .env.example
├── go.mod
└── README.md
```

## Layering (per bounded context)

```text
transport/http (Gin)  →  application  →  domain ports
                                              │
                                 infrastructure/postgres, infrastructure/hospitala
```

| Layer | May do | Must not do |
| ----- | ------ | ----------- |
| **transport/http** | Bind/validate, call the application service, map errors to HTTP | SQL, HIS HTTP, business rules |
| **application** | Auth rules, hospital scope, upsert orchestration | Import Gin types; talk to DB drivers or the HIS transport package directly |
| **infrastructure/postgres** | Parameterized SQL for owned tables | Call HIS; import handlers or the application service |
| **infrastructure/hospitala** | HTTP to Hospital A + decode into the domain shape | Import handlers/repositories/the other context |
| **domain** | Aggregates, value objects, ports (`Repository`, `HISClient`) | Import Gin, SQL drivers, or an infra wire type |

## Bounded contexts

| Concern | Context | Notes |
| ------- | ------- | ----- |
| Staff create / login | `internal/staff` | `UNIQUE (username, hospital)` |
| Patient search | `internal/patient` | Always filter by JWT `hospital` |
| HIS lookup | `patient/infrastructure/hospitala`, called only from `patient/application` | Populates local patients via upsert |
| Auth | `internal/middleware` | JWT claims: `sub`, `hospital` |

## Design principles

1. One bounded context per top-level package (`staff`, `patient`); each is a small hexagon (domain → application → infrastructure/transport).
2. Thin transport handlers, application services orchestrate use cases; rich domain model under `<context>/domain/`.
3. Repositories own SQL; ports declared in `domain` so tests can inject fakes.
4. HIS I/O isolated in `patient/infrastructure/hospitala` (anti-corruption layer) — it returns domain types, never its own wire shape.
5. Migrations are an explicit step (not silent on app boot).
6. Full DDD: value objects (`hospital.Code`, `Gender`, …) and aggregate factories live in each context's `domain/`; `internal/shared` holds only what both contexts truly need in common. See [D-013](../knowledge-base/decisions.md) for why this replaced the earlier flat "light DDD" layout.

## Docker / ops entrypoints

| Artifact | Role |
| -------- | ---- |
| `Dockerfile` | Multi-stage build of `cmd/api` and `cmd/mockhis`, non-root |
| `docker-compose.yml` | nginx + api + postgres + migrate (one-shot) + hospital-a-mock |
| `Makefile` | `make up` / `down` / `logs` / `certs` / `migrate` / `test` |
| `nginx/` | Rate limit, TLS, `X-Request-ID` forwarding |

See also root [README.md](../../README.md) and [NGINX.md](../../NGINX.md).
