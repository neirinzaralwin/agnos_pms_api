# Hospital Middleware API

Go + Gin hospital middleware for staff authentication and **hospital-scoped** patient search. Patient records are stored in PostgreSQL in a schema compatible with Hospital A HIS, and can be populated via HIS lookup (Compose includes a mock Hospital A upstream for local demos).

**Stack:** Go 1.24 · Gin · PostgreSQL 16 · Nginx · Docker Compose · golang-migrate

**Documentation:** [docs/planning/Documentation.docx](docs/planning/Documentation.docx) — project planning (structure, API spec, ER diagram).

---

## How the patient flow works

Staff authenticate, then work with patients. **Hospital A HIS** (Hospital Information System) is the source of patient data. This API keeps a **local copy** in Postgres, scoped to each staff member's hospital.

The same middleware scales to **Hospital A, B, C, …**. One API, one database — rows are tagged by hospital, and each staff member only sees their own hospital's data (from the JWT, never the request body).

```mermaid
flowchart LR

  subgraph staff ["Staff"]
    direction TB
    HA([Hospital A])
    HB([Hospital B])
    HC([Hospital C …])
  end

  subgraph mw ["★ Middleware — the control plane"]
    direction TB
    AUTH[Auth + JWT hospital claim]
    SCOPE[Enforce hospital scope]
    IMPORT[Import from HIS]
    SEARCH[Local search]
    AUTH --- SCOPE
    SCOPE --- IMPORT
    SCOPE --- SEARCH
  end

  HIS[Hospital A HIS]
  subgraph db ["Postgres"]
    direction TB
    T1["hospital-a rows"]
    T2["hospital-b rows"]
    T3["hospital-c rows"]
  end

  staff ==> mw
  mw <--> HIS
  mw ==> db

  style mw fill:#1a365d,stroke:#63b3ed,stroke-width:3px,color:#fff
  style AUTH fill:#2c5282,stroke:#90cdf4,color:#fff
  style SCOPE fill:#2c5282,stroke:#90cdf4,color:#fff
  style IMPORT fill:#2c5282,stroke:#90cdf4,color:#fff
  style SEARCH fill:#2c5282,stroke:#90cdf4,color:#fff
```

Staff never talk to HIS or Postgres directly. The middleware sits in the middle: authenticate, tag data by hospital, import when needed, and search only within that hospital.

### Hospital isolation (security)

Hospital A staff cannot see Hospital B's patients. The middleware takes the hospital from the **JWT**, adds it to every query, and ignores any hospital field in the request body.

```mermaid
flowchart TB

  SA([Hospital A staff])
  JWT["JWT claim\nhospital = hospital-a"]

  SA --> JWT
  JWT --> MW

  subgraph MW ["Middleware gate"]
    direction TB
    G1[Read hospital from JWT only]
    G2["SQL always includes\nWHERE hospital = jwt.hospital"]
    G1 --> G2
  end

  subgraph PG ["Postgres"]
    direction LR
    RA[("hospital-a\nrows")]
    RB[("hospital-b\nrows")]
  end

  G2 -->|"allowed"| RA
  G2 -.->|"blocked"| RB

  style MW fill:#1a365d,stroke:#63b3ed,stroke-width:3px,color:#fff
  style G1 fill:#2c5282,stroke:#90cdf4,color:#fff
  style G2 fill:#2c5282,stroke:#90cdf4,color:#fff
  style RA fill:#276749,stroke:#9ae6b4,color:#fff
  style RB fill:#742a2a,stroke:#fc8181,color:#fff
```

Same rule for Hospital B staff — they only reach `hospital-b` rows. Cross-hospital reads are not possible through the API.

### Fetch from HIS → keep a local copy

When staff look up a patient by ID, the middleware always asks **Hospital A HIS**, then **saves the result in Postgres** under the caller's hospital (upsert: insert if new, update if already stored). After that, **local search** can find the patient without calling HIS again.

```mermaid
sequenceDiagram
  actor Staff
  participant MW as Middleware
  participant HIS as Hospital A HIS
  participant DB as Postgres

  Staff->>MW: Look up patient by ID
  MW->>HIS: Fetch patient
  alt Found in HIS
    HIS-->>MW: Patient payload
    MW->>DB: Upsert under JWT hospital
    Note over DB: Not on our server yet → insert<br/>Already stored → update
    DB-->>MW: Saved
    MW-->>Staff: Return patient
  else Not in HIS
    HIS-->>MW: 404
    MW-->>Staff: 404 Not found
  end

  Staff->>MW: Search with filters
  MW->>DB: Query WHERE hospital = JWT hospital
  DB-->>MW: Matching rows
  MW-->>Staff: Results (HIS not called)
```

```mermaid
flowchart LR

  subgraph first ["First time — not on our server"]
    direction TB
    A1[Staff asks for ID] --> A2[Fetch from HIS]
    A2 --> A3[Insert into Postgres]
    A3 --> A4[Return patient]
  end

  subgraph later ["Later — already stored"]
    direction TB
    B1[Staff asks for same ID] --> B2[Fetch from HIS again]
    B2 --> B3[Update local row]
    B3 --> B4[Return patient]
  end

  subgraph search ["Local search"]
    direction TB
    C1[Staff searches filters] --> C2[Postgres only]
    C2 --> C3[No HIS call]
  end

  style A3 fill:#276749,stroke:#9ae6b4,color:#fff
  style B3 fill:#2c5282,stroke:#90cdf4,color:#fff
  style C2 fill:#276749,stroke:#9ae6b4,color:#fff
```

1. **Import** — fetch from HIS by ID → upsert locally under the caller's hospital → return.
2. **Search** — read Postgres only (HIS is not called), always limited to that hospital.

Adding another hospital is the same model: register staff with that hospital code; scoping stays automatic.

### Local mock HIS (demo data)

Compose runs `cmd/mockhis` as a stand-in for Hospital A. Lookups by id return canned patients (or forced errors) so demos work offline.

```mermaid
flowchart LR

  API[Middleware API] -->|"GET /patient/search/:id"| MOCK[Mock HIS]

  subgraph fixtures ["Fixture ids"]
    direction TB
    F1["1100700123456\nSomchai Jaidee · national id"]
    F2["A1234567\nAlice Wong · passport"]
    F3["ODDGENDER\nvalid payload, gender = X"]
  end

  subgraph reserved ["Reserved ids — failure paths"]
    direction TB
    E1["500ERROR → HTTP 500"]
    E2["BADJSON → broken JSON body"]
    E3["unknown id → HTTP 404"]
  end

  MOCK --> fixtures
  MOCK --> reserved

  style MOCK fill:#1a365d,stroke:#63b3ed,stroke-width:3px,color:#fff
  style F1 fill:#276749,stroke:#9ae6b4,color:#fff
  style F2 fill:#276749,stroke:#9ae6b4,color:#fff
  style F3 fill:#744210,stroke:#f6e05e,color:#fff
  style E1 fill:#742a2a,stroke:#fc8181,color:#fff
  style E2 fill:#742a2a,stroke:#fc8181,color:#fff
  style E3 fill:#742a2a,stroke:#fc8181,color:#fff
```

| Id              | What you get                                           |
| --------------- | ------------------------------------------------------ |
| `1100700123456` | Happy-path patient (Thai + English names, national id) |
| `A1234567`      | Happy-path patient (passport id)                       |
| `ODDGENDER`     | 200 OK but `gender` outside `M`/`F` (edge case)        |
| `500ERROR`      | Upstream 500 → API maps to **502**                     |
| `BADJSON`       | Undecodable body → API maps to **502**                 |
| anything else   | HIS 404 → API **404**                                  |

After a successful import, the same patient is in Postgres under the caller's hospital — then **local search** can find it without calling the mock again.

---

## Try the API (Postman)

Step-by-step requests (create staff → login → HIS import → local search, plus negatives) live in Postman:

1. Import [`docs/postman/Patient_Management_System_API.postman_collection.json`](docs/postman/Patient_Management_System_API.postman_collection.json) and [`docs/postman/Local.postman_environment.json`](docs/postman/Local.postman_environment.json)
2. Select the **Local** environment
3. Run the **Demo walkthrough** folder top-to-bottom (login saves `accessToken`)

Details: [docs/postman/README.md](docs/postman/README.md).

---

## Architecture

```mermaid
flowchart LR
  Client(["Staff client"])
  Nginx["Nginx"]
  API["Gin API"]
  PG[("PostgreSQL")]
  HIS["hospital-a-mock"]

  Client --> Nginx --> API
  API --> PG
  API -.-> HIS
```

---

## Design decisions (highlights)

- **Staff and patients are separate modules** — each owns its own logic and database access.
- **Two ways to get patients** — import from HIS by ID, or search what is already saved locally (search never calls HIS).
- **Search uses POST** — so sensitive filters are not stored in URLs or access logs.
- **One access token** — no refresh token; expiry is configurable.
- **Patients are unique per hospital** — same national/passport id can exist at different hospitals.
- **Schema migrations are manual** — applied as their own step, not on every API start.
- **Rate limits at the edge and in the app** — fine for a single instance; use a shared store if you run many replicas.

---

## Project layout

Each feature is a **bounded context** with the same shape. Add Hospital D staff, a new domain, or another HIS adapter without rewriting the core — drop in a parallel package and wire it in `cmd/api` + `httpapi`.

```text
cmd/
  api/                         # composition root — wire contexts + start server
  mockhis/                     # local Hospital A stub

internal/
  staff/                       # bounded context
    domain/                    #   aggregates, VOs, ports
    application/               #   use cases
    infrastructure/postgres/   #   SQL adapter
    transport/http/            #   Gin handlers + DTOs

  patient/                     # bounded context
    domain/
    application/
    infrastructure/
      postgres/                #   SQL adapter
      hospitalA/               #   HIS anti-corruption layer
      # hospitalB/             ← same pattern for another upstream
    transport/http/

  # <next-context>/            ← e.g. appointments: same 4 layers

  shared/                      # small shared kernel (hospital Code, errors, HTTP envelope)
  httpapi/                     # route registration only — no business logic
  middleware/                  # JWT, request id, recovery, rate limit
  config/  platform/           # env, DB pool, logger, infra errors

migrations/                    # versioned SQL (one folder, grows with contexts)
nginx/
docs/
  postman/
```

---

## Quick start

```bash
cp .env.example .env
make up
curl http://localhost:8080/healthz
```

That starts Postgres, migrations, mock HIS, API, and Nginx. Then use [Postman](#try-the-api-postman) to exercise the APIs.

| Port   | Service                   |
| ------ | ------------------------- |
| `8080` | Nginx → API               |
| `8443` | Nginx HTTPS (self-signed) |
| `5432` | Postgres                  |

```bash
make test   # go test ./... -race -cover
make logs
make down
```

See [NGINX.md](NGINX.md) for the edge proxy details.
