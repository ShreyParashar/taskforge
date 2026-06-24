-- +goose Up
-- TaskForge: Workflow tables for DAG-based workflow orchestration.

-- Workflows represent a group of dependent steps forming a DAG.
-- Workflow status is derived from its step statuses.
CREATE TABLE workflows (
    id UUID PRIMARY KEY,
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    input JSONB NOT NULL,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency index for workflows.
CREATE UNIQUE INDEX workflows_idempotency_idx
    ON workflows(namespace_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Workflow steps represent individual nodes in the workflow DAG.
-- Each step declares dependencies on other steps via depends_on.
-- When all dependencies succeed, the step is enqueued as a job.
CREATE TABLE workflow_steps (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    name TEXT NOT NULL,
    job_type TEXT NOT NULL,
    queue_name TEXT NOT NULL,
    payload JSONB NOT NULL,
    depends_on JSONB NOT NULL DEFAULT '[]'::JSONB,
    status TEXT NOT NULL,
    job_id UUID REFERENCES jobs(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_id, name)
);

-- Add foreign keys from jobs back to workflows now that the tables exist.
ALTER TABLE jobs
    ADD CONSTRAINT jobs_workflow_fk
    FOREIGN KEY (workflow_id) REFERENCES workflows(id);

ALTER TABLE jobs
    ADD CONSTRAINT jobs_workflow_step_fk
    FOREIGN KEY (workflow_step_id) REFERENCES workflow_steps(id);

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_workflow_step_fk;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_workflow_fk;
DROP TABLE IF EXISTS workflow_steps;
DROP TABLE IF EXISTS workflows;