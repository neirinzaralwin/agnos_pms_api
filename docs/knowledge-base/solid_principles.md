# SOLID Principles — Patient Management System API

Applied to Go + Gin in this codebase.

## Single Responsibility

- Handlers: HTTP ↔ DTO only
- Services: one bounded business concern (staff auth, patient search)
- Repositories: persistence for one aggregate (Staff or Patient)
- HIS client: Hospital A transport and decode only

## Open/Closed

- Extend via new methods/packages or new client adapters, not flag-bombing existing services
- Add a new HIS hospital behind `internal/client/<name>/` implementing a shared search interface rather than hard-coding Hospital A branches throughout patient service

## Liskov Substitution

- Test doubles must honor the same interface contracts (same error semantics for not-found, unauthorized, upstream failure)
- Repository fakes return the same model shapes as the Postgres implementation

## Interface Segregation

- Prefer small interfaces (`PatientRepository`, `HospitalAClient`) over a mega “AppDeps” used everywhere
- Handlers depend only on the service API they need

## Dependency Inversion

- Services depend on repository and HIS **interfaces**, not concrete Postgres/HTTP structs
- Wire concretes in `cmd/api` (composition root)
- Config via injected struct, not scattered env reads in domain logic

## Pragmatism

- Skip an interface when there is exactly one implementation and tests can use a lightweight fake without pain — but **do** interface the HIS client and repositories early; they are the primary test seams for this assignment
