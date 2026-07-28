# Dependency Map — Patient Management System API

## Process / infrastructure

```
Staff client ──► Nginx ──► Gin API (cmd/api) ──► PostgreSQL
                                │
                                └──► Hospital A HIS API
```

Compose services depend on each other only via network config (proxy upstream, `DATABASE_URL`). Application packages must not hardcode Compose hostnames outside `config`.

## Internal package direction

```
cmd/api
  └──► handler ──► service ──► repository ──► platform/db ──► PostgreSQL
                     │
                     ├──► client/hospitala ──► Hospital A
                     └──► platform (logger, errors)

middleware ──► (reads auth; may call staff validation helpers)
handler / engine ──► middleware
```

| From                 | May import                                                                 |
| -------------------- | -------------------------------------------------------------------------- |
| `cmd/api`            | handler, service, repository, middleware, client, config, platform, Gin    |
| `handler`            | service (interfaces), dto/model, platform errors, Gin                      |
| `service`            | repository interfaces, `client/hospitala` interface, model, platform, config |
| `repository`         | model, platform DB, SQL driver                                             |
| `client/hospitala`   | config, HIS DTOs, `net/http` (or shared HTTP helper in platform)           |
| `middleware`         | config, platform, auth claims helpers (not full patient service)           |
| `config`             | stdlib / light env libs only                                               |
| `platform`           | stdlib / third-party infra SDKs only                                       |
| `model` / `dto`      | stdlib only (no Gin, no DB drivers)                                        |

## Forbidden edges

- `handler` → `repository`
- `handler` → `client/hospitala`
- `repository` → `service` or `handler`
- `repository` → `client/hospitala`
- `platform` → `handler` / `service` / `repository`
- `model` → Gin or SQL
- Any package → another module’s unexported internals via unsafe patterns

## Third-party direction

- Prefer depending on **interfaces** defined next to the consumer (or in `service`) for repositories and HIS client so unit tests can inject fakes.
- Gin stays at the edge (`cmd/api`, `handler`, `middleware`). Domain services accept `context.Context` and plain structs.

## Mental model

```
controllers/handlers  →  services  →  repositories  →  DB
                              ↓
                         external clients (HIS)
```

Never skip the service layer for business operations.
