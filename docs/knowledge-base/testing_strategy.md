# Testing Strategy — Patient Management System API

**Unit test coverage is an explicit evaluation criterion** for this assignment, and the requirement states tests must cover "both positive and negative test scenarios" for *each* API. This document defines what that means concretely so it is not left to judgment per PR.

Related: [`coding_rules.md`](coding_rules.md) · [`go_gin_rules.md`](go_gin_rules.md) · [`security_rules.md`](security_rules.md)

---

## 1. Targets

| Scope                                        | Minimum line coverage |
| -------------------------------------------- | --------------------- |
| `internal/service/`                           | 90%                   |
| `internal/handler/`                           | 85%                   |
| `internal/middleware/`                        | 85%                   |
| `internal/repository/` (integration-tested)   | 70%                   |
| Overall module                                | **80%**               |

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1     # overall %
go tool cover -html=coverage.out -o coverage.html
```

Coverage is a floor, not a goal. A 95% suite that never asserts the hospital-scoping invariant is worse than an 80% suite that does. Do not chase the number with tests that call code without asserting behavior.

## 2. Shape of the suite

- **Unit tests (majority)** — services and handlers with fake repositories and a fake HIS client. No network, no database, millisecond runtime.
- **Handler tests** — real Gin engine via `httptest.NewRecorder()` and `http.NewRequest`, real routing and middleware, faked service layer. This catches binding tags, status codes, and route wiring that pure service tests miss.
- **Integration tests (few)** — real Postgres, behind a build tag so `go test ./...` stays fast and dependency-free:

  ```go
  //go:build integration
  ```

  Run with `go test -tags=integration ./...`. Use `testcontainers-go` or a Compose-provided test database. These are the only tests allowed to touch a real DB, and they exist mainly to prove the SQL and migrations are correct.

**Never** call the real Hospital A API from any test. The HIS client's own tests run against `httptest.NewServer` returning canned payloads.

## 3. Required cases per endpoint

Each row is a test that must exist. This is a floor, not the full list.

### `POST /staff/create`

| # | Scenario | Expect |
|---|---|---|
| 1 | Valid payload | `201`, staff persisted, response contains no password field |
| 2 | Password stored | Persisted value is a bcrypt hash, not the plaintext |
| 3 | Duplicate `(username, hospital)` | `409 CONFLICT` |
| 4 | Same username, different hospital | `201` — allowed per [D-006](decisions.md) |
| 5 | Missing required field | `400 INVALID_INPUT` |
| 6 | Password below minimum length | `400 INVALID_INPUT` |
| 7 | Malformed JSON body | `400 INVALID_INPUT` |
| 8 | Repository error | `500 INTERNAL_ERROR`, generic message, error logged |

### `POST /staff/login`

| # | Scenario | Expect |
|---|---|---|
| 1 | Valid credentials | `200`, token parses, claims carry correct `staff_id` + `hospital` |
| 2 | Wrong password | `401 UNAUTHORIZED` |
| 3 | Unknown username | `401` — **identical body and code to case 2** |
| 4 | Correct username/password, wrong hospital | `401` — identical response |
| 5 | Missing field | `400 INVALID_INPUT` |
| 6 | Token TTL | `exp` matches configured TTL |

Cases 2–4 returning indistinguishable responses is the anti-enumeration guarantee from [`security_rules.md`](security_rules.md) §2. Assert the response bodies are byte-identical.

### `GET /patient/search/:id` (HIS lookup)

| # | Scenario | Expect |
|---|---|---|
| 1 | HIS returns a patient | `200`, record upserted with the **caller's** hospital |
| 2 | Second lookup, same id | Updates the existing row, does not duplicate |
| 3 | No token | `401 UNAUTHORIZED` |
| 4 | Expired token | `401` |
| 5 | Malformed / wrong-algorithm token | `401` |
| 6 | HIS returns 404 | `404 NOT_FOUND` |
| 7 | HIS returns 500 | `502 BAD_GATEWAY` |
| 8 | HIS times out | `502`, request does not hang past the client timeout |
| 9 | HIS returns undecodable body | `502` |
| 10 | Invalid `id` format | `400`, HIS client is **never called** |
| 11 | Field mapping | All HIS fields land in the correct columns; `gender` outside `{M,F}` → `NULL` |

### `POST /patient/search` (local search)

| # | Scenario | Expect |
|---|---|---|
| 1 | Single filter match | `200`, matching patients returned |
| 2 | Multiple filters | Predicates AND together correctly |
| 3 | **Cross-hospital isolation** | Patient exists at hospital B, staff from hospital A searches an exactly-matching filter → empty result, not `403`, no leak of existence |
| 4 | Body contains `"hospital": "other"` | Ignored; results still scoped to the token's hospital |
| 5 | No filters supplied | `400 INVALID_INPUT` — no full-table dump |
| 6 | No token | `401` |
| 7 | No matches | `200` with an empty array, not `404`, and `[]` not `null` in JSON |
| 8 | Result cap | More rows than the max limit → response is capped |
| 9 | SQL metacharacters in a filter (`' OR 1=1--`) | Treated as a literal value; zero results; no error |
| 10 | Repository error | `500`, generic message |

Case 3 is the single most important test in the suite. It should be impossible to delete it without a reviewer noticing — name it explicitly, e.g. `TestPatientSearch_DoesNotReturnPatientsFromAnotherHospital`.

## 4. Conventions

- **Table-driven** with `t.Run(tc.name, ...)`. Names describe behavior, not mechanics: `returns_401_when_hospital_does_not_match`, not `test2`.
- `t.Parallel()` on tests without shared mutable state.
- Assert on **behavior and contract** — status code, error `code` field, persisted state — not on internal call counts, except where "was not called" is the actual requirement (HIS client on invalid input).
- No sleeps. Control time by injecting a clock function where TTL or expiry matters.
- Fixtures via small builder helpers (`newTestPatient(opts...)`) so a test only states the fields it cares about.
- Fakes are hand-written structs implementing the interface, with function fields for per-test overrides:

  ```go
  type fakePatientRepo struct {
      SearchFn func(ctx context.Context, hospital string, f SearchFilter) ([]model.Patient, error)
  }
  ```

  No mocking framework. Fakes must honor the same error semantics as the real implementation ([Liskov](solid_principles.md)) — if the real repo returns `ErrNotFound`, the fake does too.
- Tests live in the same package for internals; use `package foo_test` when exercising only the public surface.

## 5. Commands

```bash
gofmt -l .                       # must print nothing
go vet ./...
go test ./... -race -cover       # unit tests, no external dependencies
go test -tags=integration ./...  # requires Postgres
golangci-lint run                # when configured
govulncheck ./...
```

`-race` is required on the default run — cheap here, and it is the only thing that will catch a data race introduced by concurrent HIS lookups.

## 6. CI

A minimal GitHub Actions workflow running the commands above on push and PR is worth the twenty lines. It demonstrates the suite actually passes on a clean checkout rather than only on the author's machine, and it is the cheapest available signal against the "code quality" and "unit test coverage" criteria. Print the coverage percentage in the job output.

## 7. Definition of done for any change

- [ ] New behavior has both a positive and at least one negative test
- [ ] Coverage thresholds in §1 still met
- [ ] `go test ./... -race` passes
- [ ] `go vet` and `gofmt` clean
- [ ] Authorization-relevant changes include a cross-hospital isolation test
- [ ] No test reaches the network or a real database outside the `integration` build tag
