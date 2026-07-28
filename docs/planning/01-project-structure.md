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
│   ├── api/main.go          # Composition root: config → DB → router → http.Server
│   └── mockhis/main.go      # Local Hospital A stub for Compose demos
├── internal/
│   ├── handler/             # Gin handlers (bind → service → JSON)
│   ├── service/             # Business rules, hospital scoping, HIS orchestration
│   ├── repository/          # Postgres access (parameterized SQL only)
│   ├── model/               # Domain / persistence structs
│   ├── dto/                 # Request/response shapes (optional split)
│   ├── middleware/          # Recovery, request ID, logger, timeout, JWT auth, rate limit
│   ├── client/hospitala/    # Sole Hospital A HTTP adapter
│   ├── config/              # Env load + fail-fast validation
│   └── platform/            # DB pool, slog logger, JWT, shared errors
├── migrations/              # Versioned SQL (.up.sql / .down.sql)
├── nginx/                   # Reverse-proxy config + TLS certs (generated)
├── docs/planning/           # This planning set (interviewer-facing)
├── docker-compose.yml       # nginx + api + postgres + migrate + hospital-a-mock
├── Dockerfile               # Multi-stage: targets `api` and `mockhis`
├── Makefile
├── .env.example
├── go.mod
└── README.md
```

## Layering

```text
handler (Gin)  →  service  →  repository  →  PostgreSQL
                     │
                     └──► client/hospitala  →  Hospital A
```

| Layer | May do | Must not do |
| ----- | ------ | ----------- |
| **Handler** | Bind/validate, call service, map errors to HTTP | SQL, HIS HTTP, password policy details |
| **Service** | Auth rules, hospital scope, upsert orchestration | Import Gin types; talk to DB drivers directly |
| **Repository** | Parameterized SQL for owned tables | Call HIS; import handlers |
| **HIS client** | HTTP to Hospital A + decode | Import handlers/repositories |

## Domain packages

| Concern | Package | Notes |
| ------- | ------- | ----- |
| Staff create / login | `handler` + `service` + `repository` | `UNIQUE (username, hospital)` |
| Patient search | same | Always filter by JWT `hospital` |
| HIS lookup | `client/hospitala` used only from patient service | Populates local patients via upsert |
| Auth | `middleware` | JWT claims: `sub`, `hospital` |

## Design principles

1. One domain concern per package under `internal/`.
2. Thin handlers, fat services.
3. Repositories own SQL; interfaces at service boundaries for tests.
4. HIS I/O isolated in `client/hospitala`.
5. Migrations are an explicit step (not silent on app boot).

## Docker / ops entrypoints

| Artifact | Role |
| -------- | ---- |
| `Dockerfile` | Multi-stage build of `cmd/api` and `cmd/mockhis`, non-root |
| `docker-compose.yml` | nginx + api + postgres + migrate (one-shot) + hospital-a-mock |
| `Makefile` | `make up` / `down` / `logs` / `certs` / `migrate` / `test` |
| `nginx/` | Rate limit, TLS, `X-Request-ID` forwarding |

See also root [README.md](../../README.md) and [NGINX.md](../../NGINX.md).
