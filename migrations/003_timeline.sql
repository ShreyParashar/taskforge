-- +goose Up
-- TaskForge: Timeline events, schedules, and idempotency records.

-- Timeline events form an append-only audit log for jobs and workflows.
-- Every state transition, retry, lease expiry, and recovery is recorded.
-- This enables operators to inspect exactly what happened during execution.
CREATE TABLE timeline_events (
    id BIGSERIAL PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    workflow_id UUID,
    job_id UUID,
    event_type TEXT NOT NULL,
    message TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fetching timeline by job or workflow.
CREATE INDEX timeline_events_job_idx ON timeline_events(job_id)
    WHERE job_id IS NOT NULL;
CREATE INDEX timeline_events_workflow_idx ON timeline_events(workflow_id)
    WHERE workflow_id IS NOT NULL;
CREATE INDEX timeline_events_namespace_idx ON timeline_events(namespace_id, created_at DESC);

-- Schedules define recurring cron-based job creation.
-- The scheduler service finds due schedules and creates jobs accordingly.
CREATE TABLE schedules (
    id UUID PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    queue_id UUID NOT NULL REFERENCES queues(id),
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    job_type TEXT NOT NULL,
    payload_template JSONB NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    next_run_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(namespace_id, name)
);

-- Idempotency records provide a general-purpose idempotency ledger
-- beyond the per-table unique indexes. This supports detecting when
-- the same idempotency key is reused with a different request body.
CREATE TABLE idempotency_records (
    id UUID PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(namespace_id, key)
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS timeline_events;