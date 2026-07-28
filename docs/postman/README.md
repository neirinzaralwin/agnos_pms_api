# Postman — Patient Management System API

Importable collection and environment so anyone can exercise the APIs from [Project_Requirement.md](../../Project_Requirement.md).

## Files

| File | Purpose |
| ---- | ------- |
| [Patient_Management_System_API.postman_collection.json](Patient_Management_System_API.postman_collection.json) | Requests, folders, and basic test scripts |
| [Local.postman_environment.json](Local.postman_environment.json) | `baseUrl`, demo credentials, fixture ids |

## Setup

1. Start the stack: `cp .env.example .env && make up`
2. Open Postman → **Import** → select both JSON files above
3. Top-right environment picker → choose **Local**
4. Open the **Demo walkthrough** folder and run requests **1 → 4** in order  
   (or Collection Runner on that folder)

Login saves `accessToken` into the environment/collection variables for patient routes.

## What is covered

| Folder | Contents |
| ------ | -------- |
| Demo walkthrough | Create staff → login → HIS lookup → local search |
| Health | `/healthz`, `/readyz` |
| Staff | Create + login |
| Patient | HIS lookup (national id / passport) + local search |
| Negative cases | Short password, bad login, missing token, unknown id, HIS `500ERROR` → 502, empty search |

## Variables

| Variable | Default | Notes |
| -------- | ------- | ----- |
| `baseUrl` | `http://localhost:8080` | Nginx edge. Use `https://localhost:8443` for TLS (`-k` / disable SSL verify in Postman) |
| `username` / `password` / `hospital` | `alice` / `password12345` / `hospital-a` | Demo staff |
| `patientLookupId` | `1100700123456` | Mock HIS national id |
| `passportLookupId` | `A1234567` | Mock HIS passport |
| `accessToken` | _(set by Login)_ | Bearer JWT |

Mock HIS reserved ids for negatives: `500ERROR`, `BADJSON`.

## Alternative: OpenAPI

With the stack up and docs enabled, browse Swagger UI at `http://localhost:8080/docs/swagger` or import `docs/openapi/openapi.yaml` into Postman directly.
