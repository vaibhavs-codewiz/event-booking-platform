# Event Booking Platform

A backend API for an event booking system with two user types — **organizers**
(who manage events) and **customers** (who browse events and book tickets) —
role-based access control, background email notifications, and overbooking-safe
concurrent booking built on a modular monolith architecture in Go.

> **Status: Phase 1 — Planning & Foundation complete.**
> This phase establishes the project structure, architecture documentation,
> API and database design, environment configuration, local infrastructure
> (PostgreSQL + Redis via Docker Compose), and a minimal running application
> that loads config, connects to both datastores, and serves `GET /health`.
> Business APIs, authentication, booking logic, email, and background jobs are
> **not** implemented yet. Performance numbers are **not** claimed yet.

---

## Table of contents

- [Project overview](#project-overview)
- [Functional requirements](#functional-requirements)
- [Non-functional requirements](#non-functional-requirements)
- [Technology stack](#technology-stack)
- [Architecture overview](#architecture-overview)
- [Key architecture decisions](#key-architecture-decisions)
  - [Modular monolith](#modular-monolith)
  - [Database — PostgreSQL, pgx, sqlc](#database--postgresql-pgx-sqlc)
  - [Redis and Asynq](#redis-and-asynq)
  - [Email provider — Resend](#email-provider--resend)
  - [Authentication — JWT](#authentication--jwt)
  - [Authorization — role-based access control](#authorization--role-based-access-control)
- [Concurrency strategy (high level)](#concurrency-strategy-high-level)
- [Performance testing strategy](#performance-testing-strategy)
- [Deployment plan](#deployment-plan)
- [Project structure](#project-structure)
- [Local development](#local-development)
- [Planned API](#planned-api)
- [Database design](#database-design)
- [Verification (Phase 1)](#verification-phase-1)
- [Phase roadmap](#phase-roadmap)

---

## Project overview

The platform supports:

- **Event organizers** who create and manage events (capacity, schedule, status).
- **Customers** who browse upcoming events and book one or more seats.
- **Role-based access control** so only organizers manage events and only
  customers book, with each customer only able to cancel/view their own
  bookings.
- **Two background tasks**, run by a separate worker process:
  1. Booking confirmation email after a successful booking.
  2. Event update notification to all customers who booked tickets for that
     event (e.g. when an organizer changes venue or schedule).
- **Load testing** of concurrent booking requests with optimization based on
  the measured bottleneck.

---

## Functional requirements

- **Users & roles**
  - A user registers as either an `organizer` or a `customer`.
  - Email is unique; password is stored hashed (never plaintext).
- **Authentication**
  - Login returns a signed JWT access token.
  - Protected routes require a valid token.
- **Events (organizer)**
  - Create, read, update, cancel an event.
  - Only the organizer who owns the event may modify or cancel it.
- **Events (customer)**
  - Browse upcoming events; view a single event's details and availability.
- **Bookings (customer)**
  - Book `n` seats for an event. The booking must never exceed the event
    capacity, even under concurrent requests.
  - Cancel an existing booking (returns seats to the event).
  - View their own bookings.
- **Bookings (organizer)**
  - View all bookings for an event they own.
- **Notifications (background)**
  - Send a booking confirmation email on successful booking.
  - Send an event-update email to every customer with a confirmed booking for
    that event when the organizer changes it.
- **Health**
  - A health endpoint for deployment/monitoring.

---

## Non-functional requirements

| Area | Requirement |
| --- | --- |
| **Concurrency** | Booking must be overbooking-safe under concurrent requests. |
| **Performance** | Booking throughput and p95 latency must be measured under load and the bottleneck optimized (numbers reported after measurement). |
| **Security** | Passwords hashed with bcrypt; secrets via environment; JWT signed with a server secret; no secrets in the repo; `.env` git-ignored. |
| **Reliability** | Graceful shutdown; liveness/readiness probes; resilient email delivery via retries (worker). |
| **Maintainability** | Modular monolith with clear feature boundaries; type-safe DB access via sqlc; documented API (OpenAPI). |
| **Observability** | Structured request logging; health endpoints; (future) metrics and tracing. |
| **Deployability** | Container-friendly, horizontally scalable stateless API + worker; managed Postgres/Redis in production. |
| **Testability** | Unit tests for pure logic; integration tests against real Postgres/Redis; load tests with k6. |

---

## Technology stack

| Concern | Choice | Phase |
| --- | --- | --- |
| Language | Go | 1 |
| HTTP framework | Gin | 1 |
| Database | PostgreSQL | 1 |
| Postgres driver | pgx (v5) | 1 |
| Query codegen | sqlc | 2 |
| Cache / queue broker | Redis | 1 |
| Background queue | Asynq | 2 |
| Auth tokens | JWT (`golang-jwt`) | 2 |
| Password hashing | bcrypt (`golang.org/x/crypto`) | 2 |
| Transactional email | Resend | 2 |
| Local infra | Docker / Docker Compose | 1 |
| API docs | Swagger / OpenAPI | 2 |
| Load testing | k6 | 2+ |

### Why each major technology

- **Go** — Statically compiled, excellent concurrency primitives, fast startup
  and low memory footprint; a good fit for a stateless API and a queue worker
  that both need to scale horizontally.
- **Gin** — Lightweight, fast HTTP router with sensible middleware composition
  and a mature ecosystem. We use only the core (logging, recovery, route
  groups) to stay close to the standard library and keep dependencies lean.
- **PostgreSQL** — A robust, strict-consistency relational database. The
  booking problem is fundamentally about enforcing an invariant
  (seats ≤ capacity) under concurrency, which is exactly what a relational DB
  with row-level locks and CHECK constraints guarantees.
- **pgx** — The highest-performance, fully-featured Postgres driver for Go,
  with first-class `context` support, prepared statement caching, and a
  connection pool. Better suited than `database/sql` for this workload.
- **sqlc** — Generates type-safe Go from hand-written SQL instead of an ORM.
  This keeps SQL as the source of truth (the conditional booking UPDATE is
  reviewed and verified in SQL), avoids ORM abstraction leaks, and eliminates
  a class of runtime query errors at compile time.
- **Redis** — A fast, well-understood data store. Here it serves as the broker
  for the background task queue. Reusing one dependency for the queue keeps the
  operational surface small.
- **Asynq** — A Go task queue built on Redis with retries, scheduling,
  priorities, per-task concurrency control, and a web UI. It cleanly decouples
  the email side-effects from the request path so booking stays fast.
- **JWT** — Stateless, self-contained tokens that let the API scale
  horizontally without a shared session store. The token carries the user id
  and role; middleware verifies the signature and loads identity per request.
- **bcrypt** — The standard adaptive password hash; the cost factor is
  configurable and it is the de-facto choice for password storage.
- **Resend** — A developer-friendly transactional email API with good
  deliverability and a simple HTTP interface, ideal for low-volume
  transactional email (confirmations and notifications) without running an
  SMTP server.
- **Docker / Docker Compose** — Reproducible local infrastructure (Postgres +
  Redis) and the basis for production container images of the API and worker.
- **Swagger/OpenAPI** — A machine- and human-readable contract for the API,
  generated from annotations, useful for clients and for review.
- **k6** — A Go-based, scriptable load tester that models virtual users and
  concurrency scenarios, suited to stressing the booking endpoint and
  measuring p95 latency and throughput.

---

## Architecture overview

A **modular monolith** with two cooperating processes that share the same
codebase, sharing nothing at runtime except the infrastructure they talk to.

```
                  +-----------------+        +-----------------+
   HTTP clients ->|   API process   |        |  Worker process |
                  |  (cmd/api)      |        |  (cmd/worker)   |
                  |  Gin HTTP server|        |  Asynq consumer |
                  +--------+-------+        +--------+--------+
                           |                         |
        +------------------+-----------+   +----------+----------+
        |                              |   |                     |
        v                              v   v                     v
 +-------------+              +----------------+         +----------------+
 | PostgreSQL  |              |     Redis      |         |   Resend API   |
 | (pgxpool)   |              |  (Asynq queue) |         | (transactional |
 +-------------+              +----------------+         |     email)     |
                                                       +----------------+

  Request flow (booking, Phase 2):
    1. API authenticates the customer and validates input.
    2. API runs an atomic conditional UPDATE to reserve seats (overbooking-safe).
    3. API inserts the booking row in the same transaction and commits.
    4. API enqueues a "booking confirmation email" task on Redis (Asynq).
    5. Worker consumes the task and sends the email via Resend (with retries).
```

Both processes import the same internal feature packages; the API exposes HTTP
handlers, the worker registers Asynq handlers. The API never blocks on email
delivery — it only enqueues a task.

---

## Key architecture decisions

### Modular monolith

One deployable service organized into independent feature modules
(`auth`, `users`, `events`, `bookings`, `notifications`) inside a single
`internal/` tree, plus a separate worker binary.

- **Why not microservices:** the problem is small and tightly coupled; the
  booking flow touches users + events + bookings + notifications in one
  transaction. Splitting would add network boundaries, distributed
  transactions, and operational overhead with no benefit at this scale.
- **Why a separate worker (not a third microservice):** the worker is the same
  binary code with a different entry point (`cmd/worker`). It shares no state
  with the API at runtime — only the Redis queue. This gives clean separation
  of the fast request path from slow, retryable email side-effects while
  keeping deployment to two processes.
- **Why a monolith now:** faster to build, easier to refactor, simpler to run
  locally. The module boundaries are real (no import cycles, each feature owns
  its handlers and queries), so if a module later needs to split out, the
  seam already exists.

### Database — PostgreSQL, pgx, sqlc

- **PostgreSQL** guarantees the `tickets_booked ≤ capacity` invariant under
  concurrency using row-level locking and CHECK constraints — the core
  correctness requirement for bookings.
- **pgx** gives a pooled, `context`-aware, high-performance driver tuned for
  Postgres. We configure a bounded pool (`MaxConns`) sized later based on load
  test results.
- **sqlc** generates the data-access layer from SQL, so the actual booking
  query (the conditional UPDATE) is plain, reviewable SQL, and the Go callers
  get compile-time-checked types.

### Redis and Asynq

- Redis is used **only as the task queue broker** (not a cache and not a source
  of booking truth — Postgres remains the system of record).
- Asynq provides durable enqueue, retries with backoff, and per-task
  concurrency control. Email delivery is a slow, failure-prone side-effect
  (network, provider rate limits) that must not block or fail the booking
  request, so it is moved entirely to the worker.
- The API's only interaction with email is to enqueue a task; the worker owns
  delivery and retry semantics.

### Email provider — Resend

- Resend handles SMTP/deliverability; we send only transactional emails
  (booking confirmation, event update notification).
- Accessed over HTTP from the worker only. Failure is non-fatal to the
  booking — Asynq retries, and a permanently failing task lands in the
  dead-letter set for inspection without affecting bookings.

### Authentication — JWT

- Stateless, signed (HS256) access tokens. The token contains the user id and
  role; the signature is verified by middleware on every protected request.
- No server-side session store is needed, so the API scales horizontally
  behind a load balancer with no sticky sessions.
- The server secret comes from the `JWT_SECRET` environment variable and never
  ships in the repo.

### Authorization — role-based access control

- The user has exactly one role: `organizer` or `customer`.
- Authorization is enforced in two layers:
  1. **Role middleware** gates whole route groups (e.g. event management
     requires `organizer`).
  2. **Ownership checks in the handler/query** enforce finer rules (an
     organizer can only modify events they own; a customer can only cancel
     their own bookings).

---

## Concurrency strategy (high level)

The booking problem is, at its core: *reserve N seats for an event without ever
exceeding capacity, even when many customers book the same event at once.*

The approach is a **single atomic conditional UPDATE** on the `events` row,
executed inside the same transaction as the booking insert:

```sql
UPDATE events
   SET tickets_booked = tickets_booked + $2,
       updated_at     = now()
 WHERE id = $1
   AND capacity - tickets_booked >= $2
RETURNING tickets_booked;
```

- If the row is affected, the reservation succeeded — Postgres took a row lock
  for the duration of this one statement, so concurrent bookings serialize on
  the event row but hold the lock for the minimum time (no `SELECT ... FOR
  UPDATE` + separate update).
- If zero rows are affected, the requested seats are not available; the API
  returns a conflict (not enough remaining seats). There is no overbooking.
- The booking row is inserted in the **same transaction**; if it fails, the
  reservation rolls back. A `CHECK (tickets_booked <= capacity)` constraint is
  defense-in-depth against any code path that bypasses the guarded UPDATE.
- Cancellation reverses this within a transaction: flip the booking to
  `cancelled` and decrement `tickets_booked` by the booking's `seats`.

This keeps correctness in the database (the system of record) and keeps the
critical section to a single statement, minimizing lock contention. The exact
contention/throughput characteristics will be **measured with k6** (see below)
and the pool size and query shape tuned from the results.

---

## Performance testing strategy

Load testing is performed with **k6**, focused on the concurrent-booking
scenario since that is the correctness- and contention-sensitive path.

- **Scenario:** many virtual users concurrently `POST /events/{id}/bookings`
  against a single event with limited remaining capacity.
- **Metrics captured:** requests/s (throughput), p95/p99 latency, error rate,
  and — critically — the **overbooking count** (successful bookings beyond
  capacity, which must always be **zero**).
- **Bottleneck identification:** is the limit the Postgres connection pool
  (pgxpool saturation), row lock contention on the hot event row, or the
  enqueue side? Each points to a different fix.
- **Optimization loop:** adjust pgx pool sizing / `MaxConns`, shorten the
  transaction, add/verify indexes, and re-run to confirm improvement.
- **Reporting:** results are reported only **after** measurement. No
  performance numbers are claimed in Phase 1.

---

## Deployment plan

- **Phase 1 (now):** local development with Docker Compose (PostgreSQL +
  Redis). The Go application runs on the host against these services. Not
  deployed online yet.
- **Phase 2+:**
  - Container images for the API and the worker from a single Dockerfile.
  - Deploy the API (stateless, horizontally scalable) and the worker to a
    platform such as Fly.io, Render, Railway, or AWS App Runner.
  - Use **managed** PostgreSQL (e.g. Neon, Supabase, or RDS) and managed Redis
    (e.g. Upstash or ElastiCache) in production.
  - Run migrations with a versioned tool (golang-migrate).
  - Supply all secrets (`DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`,
    `RESEND_API_KEY`, `EMAIL_FROM`) from the platform's secret store.
  - CI on the public GitHub repo: lint (`go vet`), `go test ./...`, build,
    image build/push.
  - Health endpoints (`/health`, `/health/ready`) wired into the platform's
    liveness/readiness probes.

This is a plan only — deployment is **not** complete in Phase 1.

---

## Project structure

```
cactro-backend/
├── cmd/
│   ├── api/                  # HTTP API entry point (Gin server)
│   └── worker/               # Background worker entry point (Asynq consumer)
├── internal/
│   ├── auth/                 # JWT + bcrypt (Phase 2)
│   ├── users/                # User domain (Phase 2)
│   ├── events/               # Event management (Phase 2)
│   ├── bookings/             # Booking logic + concurrency-safe reservation (Phase 2)
│   ├── notifications/        # Resend email + Asynq handlers (Phase 2)
│   ├── middleware/           # Auth + role middleware (Phase 2)
│   ├── config/               # Environment configuration
│   ├── db/                   # pgx pool, redis client, sqlc queries/generated code
│   └── server/               # Gin wiring + routes + health endpoints
├── migrations/               # Versioned SQL migrations (planned schema)
├── .env.example              # Environment template (commit this)
├── .gitignore                # Ignores .env, secrets, IDE files, binaries
├── docker-compose.yml        # Local PostgreSQL + Redis
├── sqlc.yaml                 # sqlc codegen config
├── Makefile                  # Convenience commands
├── go.mod / go.sum           # Module + dependency checksums
└── README.md
```

---

## Local development

### 1. Start infrastructure

```powershell
docker compose up -d
docker compose ps   # postgres + redis should be "healthy"
```

### 2. Configure environment

```powershell
Copy-Item .env.example .env
# Edit .env: set JWT_SECRET to a long random string, add your RESEND_API_KEY later.
# DATABASE_URL and REDIS_URL already match docker-compose.yml defaults.
```

### 3. Run the application

```powershell
go run ./cmd/api
# -> listening on :8080
# -> connected to postgres
# -> connected to redis
```

### 4. Check health

```powershell
# Liveness (no dependency checks)
Invoke-RestMethod http://localhost:8080/health

# Readiness (verifies Postgres + Redis)
Invoke-RestMethod http://localhost:8080/health/ready
```

Example `/health` response:

```json
{
  "status": "ok",
  "service": "event-booking-platform",
  "time": "2026-10-18T10:00:00Z"
}
```

Example `/health/ready` response (503 if any dependency is down):

```json
{
  "status": "ok",
  "service": "event-booking-platform",
  "time": "2026-10-18T10:00:00Z",
  "dependencies": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```

---

## Planned API

Endpoints are **planned** here; none are implemented in Phase 1. Auth
requirement legend: `Public` = no token; `Auth` = any valid token; `Role: X` =
valid token with role X.

### Health

| Method | Path | Purpose | Auth | Role |
| --- | --- | --- | --- | --- |
| GET | `/health` | Liveness — process is up | Public | — |
| GET | `/health/ready` | Readiness — Postgres + Redis reachable | Public | — |

### Authentication

| Method | Path | Purpose | Auth | Role |
| --- | --- | --- | --- | --- |
| POST | `/auth/register` | Register an organizer or customer | Public | — |
| POST | `/auth/login` | Exchange credentials for a JWT | Public | — |
| POST | `/auth/refresh` | Refresh an access token | Auth | — |
| GET | `/auth/me` | Current authenticated user | Auth | any |

### Users

| Method | Path | Purpose | Auth | Role |
| --- | --- | --- | --- | --- |
| GET | `/users/me` | Get the authenticated user's profile | Auth | any |
| PUT | `/users/me` | Update the authenticated user's profile | Auth | any |

### Events

| Method | Path | Purpose | Auth | Role |
| --- | --- | --- | --- | --- |
| GET | `/events` | List upcoming events (filter/paginate) | Public | — |
| GET | `/events/{id}` | Get a single event with availability | Public | — |
| POST | `/events` | Create an event | Auth | organizer |
| PATCH | `/events/{id}` | Update an event (triggers update notifications) | Auth | organizer (owner) |
| DELETE | `/events/{id}` | Cancel an event | Auth | organizer (owner) |

### Bookings

| Method | Path | Purpose | Auth | Role |
| --- | --- | --- | --- | --- |
| POST | `/events/{id}/bookings` | Book seats for an event (overbooking-safe) | Auth | customer |
| GET | `/bookings` | List the current customer's bookings | Auth | customer |
| GET | `/events/{id}/bookings` | List bookings for an event (owner only) | Auth | organizer (owner) |
| POST | `/bookings/{id}/cancel` | Cancel one of the customer's own bookings | Auth | customer (owner) |

### Notes

- `PATCH /events/{id}` enqueues the "event update notification" task, which
  fans out an email to every customer with a confirmed booking for that event.
- `POST /events/{id}/bookings` enqueues the "booking confirmation email" task
  after the transaction commits.
- A Swagger/OpenAPI document will be generated from annotations in Phase 2.

---

## Database design

PostgreSQL schema. Primary keys are UUID (`gen_random_uuid()`). Timestamps are
`timestamptz` defaulting to `now()`. The full DDL lives in
[`migrations/001_init.up.sql`](migrations/001_init.up.sql).

### `users`

| Field | Type | Notes |
| --- | --- | --- |
| `id` | uuid | **PK**, `gen_random_uuid()` |
| `email` | text | unique (case-insensitive index on `lower(email)`) |
| `password_hash` | text | bcrypt hash, never plaintext |
| `role` | text | `CHECK (role IN ('organizer','customer'))` |
| `name` | text | nullable display name |
| `created_at` | timestamptz | `default now()` |
| `updated_at` | timestamptz | `default now()` |

- **Indexes:** `users_email_uidx` unique on `lower(email)`.

### `events`

| Field | Type | Notes |
| --- | --- | --- |
| `id` | uuid | **PK** |
| `organizer_id` | uuid | **FK → users(id)** `ON DELETE RESTRICT` |
| `title` | text | not null |
| `description` | text | nullable |
| `venue` | text | nullable |
| `start_time` | timestamptz | not null |
| `end_time` | timestamptz | nullable |
| `capacity` | integer | `CHECK (capacity > 0)` |
| `tickets_booked` | integer | `default 0`, `CHECK (>= 0)`, `CHECK (<= capacity)` |
| `status` | text | `CHECK (status IN ('active','cancelled','completed'))` |
| `created_at` | timestamptz | `default now()` |
| `updated_at` | timestamptz | `default now()` |

- **FK:** `organizer_id → users(id)`.
- **Indexes:** `events_organizer_idx (organizer_id)`, `events_start_time_idx (start_time)`.
- **Constraints:** `capacity > 0`; `tickets_booked >= 0`; `tickets_booked <= capacity` (defense-in-depth against overbooking).

### `bookings`

| Field | Type | Notes |
| --- | --- | --- |
| `id` | uuid | **PK** |
| `event_id` | uuid | **FK → events(id)** `ON DELETE RESTRICT` |
| `customer_id` | uuid | **FK → users(id)** `ON DELETE RESTRICT` |
| `seats` | integer | `CHECK (seats > 0)` |
| `status` | text | `CHECK (status IN ('confirmed','cancelled'))`, default `confirmed` |
| `created_at` | timestamptz | `default now()` |
| `updated_at` | timestamptz | `default now()` |

- **FKs:** `event_id → events(id)`, `customer_id → users(id)`.
- **Indexes:** `bookings_event_idx (event_id)`, `bookings_customer_idx (customer_id)`.
- **Constraints:** `seats > 0`; one active booking per customer per event via a
  **partial unique index** `bookings_event_customer_active_uidx` on
  `(event_id, customer_id) WHERE status = 'confirmed'` (cancelling frees the
  slot for a new booking).

### Relationships

```
users 1───* events      (an organizer owns many events)
users 1───* bookings    (a customer makes many bookings)
events 1──* bookings    (an event has many bookings)
```

### How capacity is preserved under concurrency

See [Concurrency strategy](#concurrency-strategy-high-level). Reservation is a
single atomic conditional `UPDATE ... WHERE capacity - tickets_booked >= $2`,
run in the same transaction as the booking insert. The `CHECK
(tickets_booked <= capacity)` constraint is the last line of defense.

---

## Verification (Phase 1)

Foundation verified with:

```powershell
go test ./...       # unit tests (config + health helpers)
go build ./...      # all packages compile
go run ./cmd/api    # server starts, connects to Postgres + Redis
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/health/ready
docker compose up -d
docker compose ps
```

Expected: tests pass; the server reports it connected to Postgres and Redis;
`/health` returns `{"status":"ok",...}` and `/health/ready` reports both
dependencies as `ok`. (See the "Verification" section of the Phase 1 summary for
what was actually run in this environment.)

---

## Phase roadmap

- **Phase 1 (this phase):** project structure, architecture documentation, API
  and database design, environment config, local Docker Compose (Postgres +
  Redis), minimal Go app with config loading, datastore connections, and
  `/health`. ✅
- **Phase 2:** migrations tooling, sqlc queries, auth (JWT + bcrypt), role
  middleware, users, events CRUD, overbooking-safe bookings, Asynq queue +
  worker, Resend email tasks, Swagger/OpenAPI.
- **Phase 3:** k6 load tests for concurrent bookings, bottleneck measurement
  and optimization, then re-measurement.
- **Phase 4:** containerize API + worker, CI, deploy to a hosted platform with
  managed Postgres/Redis, wire health probes.