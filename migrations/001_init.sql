-- +goose Up
-- TaskForge: Core tables for namespaces, queues, jobs, and job attempts.

-- Namespaces represent tenants, teams, or logical environments.
-- All resources (queues, jobs, workflows) are scoped by namespace.
CREATE TABLE namespaces (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Queues group jobs by execution context.
-- Workers subscribe to specific queues. Each queue can have its own
-- concurrency and rate limits.
CREATE TABLE queues (
    id UUID PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    name TEXT NOT NULL,
    concurrency_limit INTEGER NOT NULL DEFAULT 100,
    rate_limit_per_second INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(namespace_id, name)
);

-- Jobs are the core durable unit of work.
-- Each job belongs to a namespace and queue, and tracks its own
-- retry state, lease information, and scheduling.
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    queue_id UUID NOT NULL REFERENCES queues(id),
    workflow_id UUID,
    workflow_step_id UUID,

    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    idempotency_key TEXT,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    attempt_count INTEGER NOT NULL DEFAULT 0,

    locked_by TEXT,
    lease_token UUID,
    locked_until TIMESTAMPTZ,

    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency index: prevents duplicate jobs within a namespace.
-- Only applies when an idempotency_key is provided.
CREATE UNIQUE INDEX jobs_idempotency_idx
    ON jobs(namespace_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Ready jobs index: used by workers to find leasable jobs efficiently.
-- Orders by priority (desc) then creation time (asc) for fair scheduling.
CREATE INDEX jobs_ready_idx
    ON jobs(queue_id, status, run_at, priority DESC, created_at ASC)
    WHERE status = 'queued';

-- Expired lease index: used by the reaper to find stale running jobs.
CREATE INDEX jobs_expired_lease_idx
    ON jobs(status, locked_until)
    WHERE status = 'running';

-- Workflow association index: find all jobs in a workflow.
CREATE INDEX jobs_workflow_idx
    ON jobs(workflow_id)
    WHERE workflow_id IS NOT NULL;

-- Job attempts record every execution attempt for audit and debugging.
-- Each attempt tracks which worker ran it, the lease token used,
-- and the outcome (success/failure with error and duration).
CREATE TABLE job_attempts (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id),
    attempt_no INTEGER NOT NULL,
    worker_id TEXT NOT NULL,
    lease_token UUID NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    error TEXT,
    duration_ms INTEGER,
    UNIQUE(job_id, attempt_no)
);

-- +goose Down
DROP TABLE IF EXISTS job_attempts;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS queues;
DROP TABLE IF EXISTS namespaces;