# Knowledge Base — Patient Management System API

Authoritative rules for AI agents (Claude, Cursor, Copilot, etc.) working on this codebase. The root `.cursorrules`, `.cursor/rules/project.mdc`, and `CLAUDE.md` point to this directory.

**Product requirements:** [Project_Requirement.md](../../Project_Requirement.md)

## Read These In Order

1. [decisions.md](decisions.md) — **pinned technical decisions.** Read first; it settles the questions the other docs assume are already answered.
2. [architecture_overview.md](architecture_overview.md) — hospital middleware topology, domains, auth, HIS integration.
3. [folder_structure.md](folder_structure.md) — where every new file goes.
4. [module_boundaries.md](module_boundaries.md) — public HTTP surface vs private `internal/` packages.
5. [dependency_map.md](dependency_map.md) — allowed import directions.
6. [coding_rules.md](coding_rules.md) — naming, handlers, services, repos, errors, logging.
7. [go_gin_rules.md](go_gin_rules.md) — Go language and Gin framework conventions.
8. [security_rules.md](security_rules.md) — PII, authorization invariant, auth hardening, input safety.
9. [observability_and_ops.md](observability_and_ops.md) — health, shutdown, timeouts, logging, Docker, migrations.
10. [testing_strategy.md](testing_strategy.md) — coverage targets and the required test cases per endpoint.
11. [solid_principles.md](solid_principles.md) — SOLID applied to Go interfaces and Gin layers.

## The One Rule That Outranks The Rest

> **Every patient record returned to a caller must belong to the same hospital as that caller's JWT claim.**

Enforced in the patient service, sourced only from the authenticated context, never from a request field. Full statement in [security_rules.md](security_rules.md) §1; regression test required per [testing_strategy.md](testing_strategy.md) §3.

## Hard Rules (apply on every change)

- Consult [`decisions.md`](decisions.md) before re-deciding anything it already pins.
- Follow the architecture exactly as defined in `architecture_overview.md`.
- Respect module boundaries and dependency direction from `dependency_map.md`.
- Follow naming conventions, error handling, and logging from `coding_rules.md`.
- Follow Go and Gin conventions from `go_gin_rules.md`.
- Do not introduce cross-layer coupling (handler → repository skip, handler → HIS client, etc.).
- Apply SOLID principles when modifying or generating types and packages.
- Follow the folder placement strategy from `folder_structure.md`.
- Prefer improving existing structure rather than introducing new patterns.
- Never log PII or secrets; never build SQL by string concatenation.
- Every new behavior ships with a positive **and** a negative test.
- Run `gofmt`, `go vet ./...`, `go test ./... -race -cover`, and `govulncheck ./...` (and `golangci-lint run` when configured) before finishing changes.

## When Implementing Features

- Place files only in the correct architectural layer.
- Reuse existing abstractions where possible:
  - **Config / DB / logger / errors:** `internal/config`, `internal/platform`
  - **Auth:** JWT middleware in `internal/middleware`; staff hospital claim on context
  - **HIS:** `internal/client/hospitala` only — never call Hospital A HTTP from handlers
- Avoid duplicate logic.
- Keep HTTP binding/response mapping in **handlers** only.
- Keep business / orchestration logic in **services** only.
- Keep SQL and persistence in **repositories** only.
- Thread `context.Context` through every service, repository, and client call.

## Definition of Done

A change is finished when all of the following hold:

- [ ] Code sits in the correct layer, imports respect [`dependency_map.md`](dependency_map.md)
- [ ] Positive and negative tests added; coverage thresholds in [`testing_strategy.md`](testing_strategy.md) §1 still met
- [ ] Security checklist in [`security_rules.md`](security_rules.md) §12 passes for the change
- [ ] Ops checklist in [`observability_and_ops.md`](observability_and_ops.md) §12 still passes
- [ ] `gofmt` / `go vet` / `go test -race` / `govulncheck` clean
- [ ] Any decision made along the way that wasn't already pinned is recorded in [`decisions.md`](decisions.md)

## When a Request Conflicts With These Rules

1. **Explain the conflict first** — say which rule is being violated and why.
2. **Propose a compliant alternative** that achieves the user's intent.
3. **Implement the compliant solution** unless the user explicitly overrides.
