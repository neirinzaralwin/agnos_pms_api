# Folder Structure — Patient Management System API

Authoritative guide for **where new files go**. This is the target layout for the greenfield Go + Gin service.

## Design principles

1. **One domain concern per package.** Staff, patient, and HIS client stay separate under `internal/`.
2. **`internal/` is private.** Nothing outside the module may import these packages (Go visibility + convention).
3. **Thin handlers, fat services.** Gin handlers bind and respond; services own business rules.
4. **Repositories own SQL.** One repository writer per table/aggregate.
5. **HIS client is isolated.** Only `internal/client/hospitala` talks to Hospital A over HTTP.
6. **`platform/` is last resort.** Shared code only when ≥ 2 domains need it.
7. **No new top-level folders** without updating this doc and `architecture_overview.md`.

When in doubt: add to the **smallest existing package** that owns the concept.

## Repository Root

```
patient_management_system_api/
├── cmd/
│   └── api/
│       └── main.go              # process entry; wire deps; start Gin
├── internal/
│   ├── handler/                 # Gin HTTP handlers
│   ├── service/                 # business orchestration
│   ├── repository/              # Postgres access
│   ├── model/                   # domain structs / DB models
│   ├── dto/                     # request/response shapes (optional split)
│   ├── middleware/              # JWT auth, request ID, recovery
│   ├── client/
│   │   └── hospitala/           # Hospital A HIS HTTP adapter
│   ├── config/                  # env loading and validation
│   └── platform/                # db, logger, errors, helpers
├── migrations/                  # versioned SQL migrations (.up.sql + .down.sql)
├── nginx/                       # reverse-proxy config for Compose
├── docs/
│   └── knowledge-base/          # this directory
├── .github/
│   └── workflows/ci.yml         # fmt, vet, test -race, coverage
├── docker-compose.yml           # nginx + golang + postgresql
├── Dockerfile                   # multi-stage, non-root
├── .dockerignore
├── .env.example                 # keys with placeholder values; .env is gitignored
├── Makefile                     # run, test, migrate, lint shortcuts
├── go.mod
├── go.sum
├── Project_Requirement.md
├── .cursorrules
└── CLAUDE.md
```

Do not create a parallel `pkg/` for domain logic unless something must be imported by another Go module (unlikely for this assignment).

## Package placement

| Kind of change                         | Put it here                                      |
| -------------------------------------- | ------------------------------------------------ |
| New HTTP endpoint                      | `internal/handler/` + route registration in `cmd/api` or a router setup package |
| Business rule / orchestration          | `internal/service/`                              |
| SQL / queries                          | `internal/repository/`                           |
| Domain struct / persistence model      | `internal/model/`                                |
| Request/response JSON shapes           | `internal/dto/` or next to handler if tiny       |
| Auth / JWT / hospital claim injection  | `internal/middleware/`                           |
| Hospital A HTTP calls                  | `internal/client/hospitala/`                     |
| Env / secrets / feature flags          | `internal/config/`                               |
| DB pool, logger, shared errors         | `internal/platform/`                             |
| Schema change                          | `migrations/`                                    |
| Nginx upstream / proxy rules           | `nginx/`                                         |

### Decision tree

1. New endpoint for existing domain → handler method + service method (+ repository if new query).
2. New external system → `internal/client/<name>/` with an interface consumed by services.
3. Cross-domain helper used by ≥ 2 packages → `internal/platform/`.
4. Only one domain needs a helper → keep it in that domain’s package (or unexported in service).

## Naming files

| Element        | Pattern                         | Example                    |
| -------------- | ------------------------------- | -------------------------- |
| Handler        | `<domain>_handler.go`           | `staff_handler.go`         |
| Service        | `<domain>_service.go`           | `patient_service.go`       |
| Repository     | `<domain>_repository.go`        | `staff_repository.go`      |
| Model          | `<entity>.go`                   | `patient.go`               |
| Middleware     | `<name>.go`                     | `auth.go`                  |
| HIS client     | `client.go` (+ `types.go`)      | `client/hospitala/client.go` |
| Tests          | `<file>_test.go` same package   | `staff_service_test.go`    |
| Migration      | `NNNN_description.up.sql`       | `0001_init.up.sql`         |

## Docker / Compose

- `Dockerfile` builds the `cmd/api` binary.
- `docker-compose.yml` runs **nginx**, **golang service**, and **postgresql**.
- App connects to Postgres via Compose service DNS; Nginx proxies to the Go service port.

## Forbidden placements

- SQL inside `handler/`
- `net/http` calls to Hospital A outside `internal/client/hospitala`
- Domain types duplicated in handlers and repositories without a shared `model`/`dto`
- Secrets committed under the repo (use env / Compose secrets)
