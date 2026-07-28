# Claude Instructions — Patient Management System API

Go + Gin hospital middleware. Staff authenticate against their own hospital and search patients; patient records are sourced from the Hospital A HIS and stored locally in Postgres.

## Read Before Every Edit

Start at [docs/knowledge-base/README.md](docs/knowledge-base/README.md) — it indexes everything below.

Consult pinned technical decisions in docs/knowledge-base/decisions.md before re-deciding anything
Follow project architecture defined in docs/knowledge-base/architecture_overview.md
Follow coding standards defined in docs/knowledge-base/coding_rules.md
Follow Go and Gin rules defined in docs/knowledge-base/go_gin_rules.md
Follow security and PII rules in docs/knowledge-base/security_rules.md
Follow operational rules (health, shutdown, timeouts, Docker) in docs/knowledge-base/observability_and_ops.md
Follow test requirements in docs/knowledge-base/testing_strategy.md
Respect dependency direction from docs/knowledge-base/dependency_map.md
Respect module boundaries from docs/knowledge-base/module_boundaries.md
Apply SOLID principles from docs/knowledge-base/solid_principles.md
Follow folder placement strategy from docs/knowledge-base/folder_structure.md

## Non-Negotiables

- Every patient query filters by the hospital from the **authenticated JWT claim**, never from a request field
- Never log passwords, tokens, or full national/passport IDs
- Parameterized SQL only — never `fmt.Sprintf` into a query
- Structured logging via `log/slog` — no `fmt.Println` / `log.Print` in committed application code
- `context.Context` threaded through every service, repository, and client call
- Every new behavior ships with a positive **and** a negative test

## Before Finishing Any Change

```bash
gofmt -w . && go vet ./... && go test ./... -race -cover && govulncheck ./...
# golangci-lint run   (when configured)
```

Product requirements: [Project_Requirement.md](Project_Requirement.md)

---

## Graphify (future)

This project does not yet have a graphify knowledge graph. When `graphify-out/graph.json` exists, run `graphify query` / `path` / `explain` before broad code exploration. Until then, use Read/Grep/Glob normally.
