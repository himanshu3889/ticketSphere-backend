# 🎟️ TicketSphere — Backend

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql&logoColor=white)
![Redis Stack](https://img.shields.io/badge/Redis_Stack-Lua%20%C2%B7%20Bloom-DC382D?logo=redis&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-Topic%20%2B%20DLX-FF6600?logo=rabbitmq&logoColor=white)
![MongoDB](https://img.shields.io/badge/MongoDB-Projections-47A248?logo=mongodb&logoColor=white)
![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Traces-000000?logo=opentelemetry&logoColor=white)

**TicketSphere** is a backend system that simulates the highly concurrent ticket booking system.

The goal of this project is not feature completeness, but **architectural correctness under stress**: flash-sale traffic spikes, high-contention seat races, and a payment workflow that must be exactly-once even when everything retries — while keeping performance predictable and latency low.

It explores practical distributed-systems patterns: **saga choreography, atomic Redis Lua locking with TTL, three-layer idempotency, Bloom-filter edge protection, dead-letter queues with replay-safe consumers, and end-to-end distributed tracing across message-queue boundaries**.

![Ticket Booking Architecture](https://drive.google.com/thumbnail?id=1bw8OvC7mtmbmXObGWa0O4OcRS9qYANUk&sz=w5000)
> **Excalidraw file** [Discore Exaclidraw](https://drive.google.com/file/d/1mKn2jBHAb-XavylMvmGGI5CjXSuYsQ-f/view?usp=drive_link).



---

## 📖 Contents

- [The Hard Problems](#-the-hard-problems)
- [Architecture at a Glance](#%EF%B8%8F-architecture-at-a-glance)
- [The Core Problem: Two Users, One Seat](#-the-core-problem-two-users-one-seat)
- [Technology Choices — the "Why"](#-technology-choices--the-why)
- [Failure Modes](#-failure-modes)
- [Load Testing: Evidence, Not Vibes](#-load-testing-evidence-not-vibes)
- [Quick Start](#-quick-start)
- [Design Philosophy](#-design-philosophy)

---

## The Hard Problems

Four problems make ticketing a genuinely interesting systems exercise — each one maps to a section of this README:

1. **Never sell the same seat twice.** Thousands of concurrent requests for *the same row of seats* must resolve to exactly one winner, in milliseconds, without a global lock.
2. **Exactly-once… when everything retries.** Clients double-click, gateways redeliver webhooks, consumers re-drive dead letters. Every layer must tolerate being called twice.
3. **Payments are asynchronous and untrusted.** A payment completes via an external webhook, possibly late, duplicated, or forged — while someone else's money is attached. 
4. **You can't fix what you can't see.** "It feels slow" is not an engineering statement. Every claim in this README is traceable to a metric, a trace, or a load-test check. 

---

## Architecture at a Glance

Three Go services behind a declarative edge gateway. The synchronous hot path lives in Redis; Postgres owns the truth; RabbitMQ moves state changes between services; MongoDB serves read projections.


| Component | Runtime | Responsibility |
|---|---|---|
| **Goma Gateway** | Docker, `:7777` → `:8080` | JWT verification, Redis-backed per-IP & per-user rate limits with ban windows, injects `X-Gateway-Secret`, health-checks every route |
| **Inventory** | Go/Gin, `:7010` | Catalogue CRUD and seat-availability reads (venues, events, sections, seats) |
| **Booking** | Go/Gin, `:7020` | Sign-up/sign-in (JWT issue), the atomic seat-locking hot path, booking saga inbox, Postgres + Mongo projection consumers |
| **Payment** | Go/Gin, `:7030` | Payment initiation against a mock gateway, HMAC-verified webhook intake, payment consumers |


---

## The Core Problem: Two Users, One Seat

A seat hold is a check-then-set across **multiple keys** (one per seat, an idempotency key, a user hash). Any non-atomic sequence has a race window. Doing it in SQL means row locks held across a network round-trip (including price lookup) — exactly what dies first under flash-sale load.

### The Booking Saga


### The Hot Path: One Atomic Lua Script

The check-and-set for up-to-10 seats runs as a **single Lua script inside Redis** - Lua execution is atomic, so there is no interleaving between "check" and "lock," no matter how many clients race:


Why this shape matters:

- **Check-all-then-lock-all inside one script** → partial holds are impossible; a 409 means *zero* side effects.
- **Locks carry TTLs (15 min)** → a crashed client can never wedge inventory forever; holds self-release exactly when the payment window closes.
- **`HEXPIRE`** sets a TTL on a *single hash field* — each booking in a user's "latest bookings" hash expires independently, no key-per-booking sprawl.
- The lock **value** is `userId:bookingId`, so ownership is auditable and future release can be compare-and-delete.

### Why Redis-first (and why the SQL versions still exist)

`booking/store` also contains `BookSeatsOptimistic` (conditional `UPDATE ... WHERE version = ?`) and `BookSeatsPessimistic` (`SELECT ... FOR UPDATE`) implementations. They remain in-tree deliberately: they document the evolution, and let the load harness A/B strategies.

| Strategy | Behaviour under 200 VUs fighting for 100 hot seats |
|---|---|
| Pessimistic SQL | Row locks serialize every racer; lock wait dominates p99; contention on a hot row caps throughput |
| Optimistic SQL | Losers pay a full DB round-trip + retry storm on hot rows |
| **Redis Lua (chosen)** | Losers pay one in-memory script, ~µs; DB never sees losing traffic; 409 is the *expected* outcome, not an error |

The DB stays authoritative — it just never touches contention it would lose.

---

## Technology Choices — the "Why"

### Why Go?
Booking is a concurrency problem: thousands of simultaneous "hold this seat" requests, most of which must lose cheaply.
- **Goroutines + channels** make per-request spans and queue consumer pools (prefetch-bounded workers) cheap.
- **`context` everywhere** — every external call (Redis, Postgres, RabbitMQ publish) carries a deadline, so a stalled dependency cannot cascade into thread starvation.
- Predictable latency and low memory per connection compared to managed runtimes.

### Why PostgreSQL?
Source of truth for users, catalogue, bookings, payments and other relational data.
Having ACID property.

### Why Redis Stack?
One process covers five jobs that would otherwise be five systems: **distributed locks with TTL** (Lua), **idempotency store**, **cache**, **Bloom filters** (edge protection), and **rate-limit state** for the gateway. All hot-path decisions happen in-process, in microseconds, so the API stays responsive while Postgres stays calm.

### Why RabbitMQ?
 Require a fast in memory message broker, supports dead letter queue, message ack, topic routing and operation simplicity.
 Publishing is fire-and-forget for latency; delivery guarantees live in the consumers (idempotent handlers + DLQ), which is where they're testable.

### Why MongoDB?
Stores the rich JSON seat maps for every single event.
As redis is costlier can't store the map view of seats in the redis always, so to make it little inexpensive we have to use something which gives the full map view fast and update map view faster.


### Why Goma Gateway?
Simple opensource gateway,
Declarative YAML edge: JWT auth, rate limits with **ban windows** (repeat offenders are banned, not just throttled), per-route health checks, and secret injection — without writing (and debugging) a bespoke gateway.

### Why OpenTelemetry + Tempo, Prometheus, Loki, Grafana?
Because "it's slow" must decompose into *where*. Traces follow requests **through HTTP, Rabbitmq, Postgres and across RabbitMQ** (context is injected into AMQP headers and extracted by consumers — one trace from `POST /booking` to the Postgres projection). Metrics answer "how much" (including Bloom-filter verdict counts and consumer success/failure); Loki answers "show me the exact log lines," wired in via a logrus hook and explorable next to the metric that pointed at it.

---

## Failure Modes

What the system does when things go wrong — the part that usually gets hand-waved:

| Failure | System behaviour |
|---|---|
| User double-clicks "Book" / client retries | Redis idempotency hit returns the original `PENDING` booking — no duplicate locks, no duplicate charge path |
| Two users race the same seat | Lua check-and-lock is atomic; loser gets `409` naming the seat; zero partial side effects |
| Booking service crashes between lock and publish | No booking row exists; locks self-release in ≤ 15 min. |
| Gateway redelivers a webhook | HMAC verified → republished; payment-state handlers and projections are idempotent, so duplicates are absorbed |
| Consumer hits a bug / schema drift | Message is nacked to the DLQ with exact-once retention; re-drive after fix no-ops via `ON CONFLICT` upsert |
| Flood of lookups for non-existent IDs | Bloom returns "definitely not" → rejected before Postgres; verdicts visible as metrics |
| Abuse at the edge | Per-IP / per-user limits trip; 5 violations → timed ban; enforced in Redis across gateway replicas |
| Direct request to a service (bypassing gateway) | Rejected: missing `X-Gateway-Secret` → `502 Gateway bypass` |
| Redis unavailable mid-sale | Booking fails closed (500). Reads degrade rather than fail. Trade-off stated in Limitations |
| Hot key expiry during a spike | Not applicable to locks (TTLs are the protocol); caches sit behind Bloom guards so misses don't stampede the DB |

---

## Load Testing: Evidence, Not Vibes

Sustaining 3,000 bookings/sec with p50 = 1 ms / p95 = 6 ms / p99 = 205 ms end-to-end latency, 0 errors and 0 double-bookings across 90K concurrent requests, on 10 core 4.8Ghz cpu.

![Ticket Booking Load Test](https://drive.google.com/thumbnail?id=1516E0iTAUTuWYo8Dqzpi16txryDv3vqD&sz=w5000)

---

## 🚀 Quick Start

**Prerequisites:** Go 1.25+ · Docker & Docker Compose · [`migrate` CLI](https://github.com/golang-migrate/migrate) (`-tags postgres`) · (optionally) Air, k6

```bash
# 1) Create .env at the repo root — consumed by docker compose, the Makefile,
#    the gateway renderer, and every service config. Key groups:
#    POSTGRES_* · MONGODB_* · REDIS_* · RABBITMQ_* · JWT_SECRET/ISSUER/AUDIENCE
#    GATEWAY_SECRET · GATEWAY_PORT · APP_NAME/APP_ENV/APP_LOG_LEVEL
#    OTEL_ENDPOINT · LOKI_HOST/LOKI_PORT · PROMETHEUS_USER/PASSWORD/PORT
#    GRAFANA_ADMIN_PASSWORD/GRAFANA_PORT · RATE_LIMIT_PER_MINUTE
#    PAYMENT_GATEWAY_BASE_URL/WEBHOOK_URL/WEBHOOK_SECRET
#    INVENTORY_API_TARGET · BOOKING_API_TARGET · PAYMENT_API_TARGET

# 2) Infrastructure: datastores + monitoring + RabbitMQ
make up-sys

# 3) Edge gateway (renders gateway/goma.yml with envsubst, then starts Goma)
make up-gateway

# 4) Schema + Bloom filters
make migrate-post-up
make migrate-redis-up-all

# 5) Services — three terminals (or: go run ./cmd/<name>)
make air-inventory      # :7010
make air-booking        # :7020
make air-payment        # :7030
```

**Endpoints & consoles**

| What | Where |
|---|---|
| Public API (via gateway) | `http://localhost:${GATEWAY_PORT}` (e.g. `:7777`) |
| RabbitMQ management UI | `http://localhost:15672` |
| Grafana | `http://localhost:${GRAFANA_PORT}` (default `:9300` → container `3000`) |
| Prometheus | `http://localhost:${PROMETHEUS_PORT}` (default `:9090`) |
| Tempo / Loki / OTel | `:3200` / `:3100` / `:4317` |

---


## Design Philosophy

Correctness under load before features. Isolation of responsibilities over cleverness. Explicit failure handling over hopeful retries. And every architectural claim in this document is either a diagram you can trace to a file, or a number the load rig can reproduce.

Feature completeness is intentionally secondary: the goal was never to build *a* ticketing app, but to build one that stays consistent when the on-sale goes viral.

---

*Created by [Himanshu](https://github.com/himanshu3889)*
