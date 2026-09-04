# Agent Instructions

These instructions apply to the whole repository.

## Scope and workflow

- The active migration target is the Go backend. The `frontend/` directory is
  intentionally out of scope until the Go services and their API contracts are
  ready.
- Backward compatibility with the Python API, the old database schema, and old
  migrations is not required. This is a pet project with no production users;
  a local database reset is an accepted migration step.
- Read the relevant code and the migration plan before changing it. Keep each
  change within one migration phase or one service boundary.
- Use `rg`/`rg --files` for searching and `apply_patch` for manual edits.
- Do not add generated binaries, dependency caches, local `.env` files, dumps,
  certificates, or other machine-local artifacts.
- Prefer small commits that leave the repository buildable. State/checkpoint
  commits are allowed when explicitly requested.

## Go toolchain and libraries

- Use the Go version declared by the repository workspace.
- HTTP routing: `github.com/go-chi/chi/v5`.
- Internal RPC: use gRPC only where typed service-to-service calls or streaming
  provide a clear benefit; do not add gRPC as a second transport to every
  endpoint.
- PostgreSQL: `github.com/jackc/pgx/v5` and `pgxpool`; do not introduce an ORM.
- SQL construction: `github.com/Masterminds/squirrel` for dynamic queries;
  simple stable queries may use explicit SQL strings.
- Migrations: `github.com/pressly/goose/v3`. Every service owns its migration
  directory and migration version table.
- Tests: `github.com/stretchr/testify` for assertions, mocks, and suites where
  they improve readability.
- Redis: `github.com/redis/go-redis/v9` for cache, rate limiting, and ephemeral
  realtime state.
- Use the standard library where it is sufficient (`log/slog`, `context`,
  `net/http`, `crypto`, and `encoding/json`).
- All exported Go packages, types, functions, and methods must have GoDoc
  comments in Russian, starting with the documented identifier. Add Russian
  comments for non-obvious private logic as well; do not comment trivial code.
- Protobuf contracts live under `api/proto/` and are generated with `buf` when
  a service uses gRPC. Generated Go and gateway code must be reproducible from
  the checked-in `.proto` files and `buf.gen.yaml`.
- HTTP handlers use Swagger/OpenAPI annotations; generated OpenAPI output is
  produced through `task generate:swagger`.

## Service and layer boundaries

Each service follows this dependency direction:

```text
transport → application → domain
     ↓          ↓
 adapters  ports/interfaces
```

- `domain` contains entities, value objects, and pure policies. It must not
  import chi, pgx, Redis, Kafka, S3, or transport types.
- `application` contains actor-visible use cases and DTOs. It coordinates ports
  and owns transaction boundaries through an explicit unit-of-work/transaction
  port when a scenario needs several writes.
- Use explicit DTOs and mappers at every layer boundary. Transport request and
  response DTOs must not be domain entities; application input/output DTOs must
  not expose pgx rows or adapter models; adapters map database rows to domain
  objects and application maps domain objects to output DTOs.
- `ports` contains interfaces owned by the consumer/application layer.
- `adapters` contains PostgreSQL, Redis, Kafka, object storage, and external
  service implementations.
- `transport/http` contains chi handlers, request decoding, response encoding,
  authentication middleware wiring, and HTTP error mapping only.
- Constructors in `transport/http` accept application services or ports, never
  concrete `pgxpool`, Redis, Kafka, or other infrastructure clients.
- `cmd/<service>` contains composition and process lifecycle, not business
  logic.
- Services must not import another service's internal packages. Cross-service
  communication goes through HTTP, WebSocket, events, or explicitly shared
  contracts only.
- Avoid global mutable state, service locators, and hidden dependencies. Wire
  dependencies explicitly in `cmd`.

## PostgreSQL schema ownership

- One PostgreSQL instance may host all services, but every service owns a
  dedicated PostgreSQL schema. Table names must always be schema-qualified in
  migrations and queries.
- Initial ownership is documented in `docs/go-migration-plan.md`.
- Do not create foreign keys between service schemas. Store external entity IDs
  and validate them through a service call or event. This keeps later database
  extraction possible.
- A service may use foreign keys inside its own schema, with explicit
  `ON DELETE` behavior and indexes for every lookup path.
- Never edit an applied goose migration. For this pet project, reset the local
  database and create a new clean migration chain when the model changes
  materially.
- Migration SQL must be deterministic, reviewable, and safe to run in a clean
  database. Seed only stable reference data.

## HTTP, caching, and rate limiting

- Handlers return a consistent JSON error envelope and map domain/application
  errors to status codes in one place.
- Every public endpoint gets request ID, structured logging, panic recovery,
  authentication where needed, and rate-limit middleware as appropriate.
- Cache only read models that are safe to serve briefly stale. Use explicit
  versioned keys, bounded TTLs, JSON serialization, and invalidation on writes.
- Prefer Redis-backed limits and cache for multi-process correctness. An
  in-memory limiter is acceptable only for a deliberately local-only endpoint
  and must be documented.
- Never cache credentials, access tokens, private messages, or permission
  decisions without an explicit owner and invalidation strategy.

## Events and background work

- Use an outbox when a database mutation must publish an event. The business
  write and outbox row are one PostgreSQL transaction.
- Publishers and consumers must be idempotent and tolerate redelivery.
- Keep HTTP API, realtime delivery, outbox publishing, and analytics consumers
  as separate processes or service commands with explicit dependencies.

## Testing and verification

- Prefer unit tests for domain policies and application use cases with fake
  ports; use integration tests for pgx repositories, Redis, Kafka, and goose
  wiring; keep a small number of end-to-end tests for critical flows.
- New behavior should have tests before or together with implementation.
- Run, as applicable:
  - `gofmt` on changed Go files;
  - `go test ./...` for the affected module or workspace;
  - `go vet ./...`;
  - migration up/down or clean-database smoke checks;
  - `docker compose config -q` after Compose changes.
- Use `Taskfile.yml` as the canonical local entry point:
  `task fmt`, `task test`, `task vet`, `task generate:swagger`,
  `task generate:proto`, `task db:bootstrap`, and `task db:migrate SERVICE=...`.
- Goose migrations must be run through the task or the equivalent explicit
  goose command; never use Alembic for the Go schemas.
- Do not weaken or delete tests just to make a migration pass. Since API
  compatibility is explicitly out of scope, replace obsolete Python tests with
  Go tests that verify the new contract.
