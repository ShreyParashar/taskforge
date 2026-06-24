# TaskForge

> A durable workflow orchestration backend for long-running, failure-prone AI, security, and backend automation workflows.

## Why This Project Exists

While working on GenAI/security automation, I observed that the difficult backend problem is not just generating output. Real automation workflows are long-running and failure-prone: internal APIs timeout, LLM providers fail or rate-limit requests, users retry submissions, workers crash midway, webhooks fail, and expensive operations need auditability and cost control.

TaskForge was built to understand how production systems execute multi-step workflows reliably under these conditions.

## What TaskForge Does

TaskForge lets developers submit jobs and multi-step workflows that are executed reliably by worker processes. It provides:

- **Durable job state** backed by PostgreSQL
- **Worker leasing** with `FOR UPDATE SKIP LOCKED` for safe concurrent execution
- **Idempotency keys** to prevent duplicate work
- **Retries with exponential backoff and jitter** to avoid retry storms
- **Dead-letter queues** for jobs that exhaust all attempts
- **Workflow DAGs** with dependency-based step execution
- **Stale-worker recovery** via a lease reaper
- **Rate limiting** (Redis-backed) to protect downstream services
- **Circuit breakers** to stop hammering failing dependencies
- **Append-only timeline** for full execution auditability
- **Structured logs, Prometheus metrics, OpenTelemetry traces**

## Architecture

```
                  ┌────────────────┐
                  │  Client / CLI  │
                  └───────┬────────┘
                          │
                          ▼
                  ┌────────────────┐
                  │   API Server   │
                  └───────┬────────┘
                          │
            ┌─────────────┴──────────────┐
            │                            │
            ▼                            ▼
    ┌────────────────┐          ┌────────────────┐
    │  PostgreSQL    │          │     Redis      │
    │  durable state │          │  rate limits   │
    └───────┬────────┘          └────────────────┘
            ▲
            │
    ┌───────┴────────┐
    │  Worker Pool   │
    └───────┬────────┘
            │
   ┌────────┴─────────┐
   │                   │
   ▼                   ▼
┌─────────────┐  ┌─────────────┐
│  Scheduler  │  │   Reaper    │
│ delayed jobs│  │ lease expiry│
└─────────────┘  └─────────────┘
```

**Design principle:** PostgreSQL owns correctness. Redis improves coordination speed but is not required for durable recovery.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22 |
| Database | PostgreSQL 16 |
| Cache/Rate Limiting | Redis 7 |
| HTTP Router | chi |
| DB Driver | pgx/v5 |
| Logging | zerolog |
| Metrics | Prometheus client_golang |
| Tracing | OpenTelemetry |
| Migrations | goose |
| Local Runtime | Docker Compose |

## Job State Machine

```
queued ──────────► running ──────────► succeeded
  │                  │
  │                  ├──────────► retry_scheduled ──► queued
  │                  │
  │                  ├──────────► dead_lettered
  │                  │
  └──► cancelled ◄───┘
```

## Quickstart

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Run database migrations
make migrate-up

# Start the API server
make run-api
```

## Project Status

🚧 **Under active development**

- [x] Phase 1: Domain models, database migrations, configuration
- [ ] Phase 2: API handlers and HTTP routes
- [ ] Phase 3: Storage/repository layer with pgx
- [ ] Phase 4: Worker leasing and heartbeat
- [ ] Phase 5: Scheduler and reaper services
- [ ] Phase 6: Rate limiting and circuit breaker
- [ ] Phase 7: Observability (logs, metrics, traces)
- [ ] Phase 8: CLI, failure simulator, tests, benchmarks

## License

MIT