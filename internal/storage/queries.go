package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"taskforge/internal/domain"
	"taskforge/internal/domain/uid"
)

// ErrNotFound is returned when a requested record does not exist in the database.
var ErrNotFound = errors.New("storage: record not found")

// --- Namespaces ---

// CreateNamespace inserts a new namespace into the database.
func (s *Store) CreateNamespace(ctx context.Context, ns *domain.Namespace) error {
	query := `
		INSERT INTO namespaces (id, name, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := s.pool.Exec(ctx, query, ns.ID, ns.Name, ns.CreatedAt)
	if err != nil {
		return fmt.Errorf("storage: failed to create namespace: %w", err)
	}
	return nil
}

// GetNamespaceByName retrieves a namespace by its name.
func (s *Store) GetNamespaceByName(ctx context.Context, name string) (*domain.Namespace, error) {
	query := `
		SELECT id, name, created_at
		FROM namespaces
		WHERE name = $1
	`
	ns := &domain.Namespace{}
	err := s.pool.QueryRow(ctx, query, name).Scan(&ns.ID, &ns.Name, &ns.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("namespace %q: %w", name, ErrNotFound)
		}
		return nil, fmt.Errorf("storage: failed to get namespace %q: %w", name, err)
	}
	return ns, nil
}

// --- Queues ---

// CreateQueue inserts a new queue into the database.
func (s *Store) CreateQueue(ctx context.Context, q *domain.Queue) error {
	query := `
		INSERT INTO queues (id, namespace_id, name, concurrency_limit, rate_limit_per_second, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.pool.Exec(ctx, query, q.ID, q.NamespaceID, q.Name, q.ConcurrencyLimit, q.RateLimitPerSec, q.CreatedAt)
	if err != nil {
		return fmt.Errorf("storage: failed to create queue: %w", err)
	}
	return nil
}

// GetQueueByName retrieves a queue by its namespace and name.
func (s *Store) GetQueueByName(ctx context.Context, namespaceID uid.ID, name string) (*domain.Queue, error) {
	query := `
		SELECT id, namespace_id, name, concurrency_limit, rate_limit_per_second, created_at
		FROM queues
		WHERE namespace_id = $1 AND name = $2
	`
	q := &domain.Queue{}
	err := s.pool.QueryRow(ctx, query, namespaceID, name).Scan(
		&q.ID, &q.NamespaceID, &q.Name, &q.ConcurrencyLimit, &q.RateLimitPerSec, &q.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("queue %q: %w", name, ErrNotFound)
		}
		return nil, fmt.Errorf("storage: failed to get queue %q: %w", name, err)
	}
	return q, nil
}

// --- Jobs ---

// CreateJob inserts a new job into the database.
func (s *Store) CreateJob(ctx context.Context, j *domain.Job) error {
	query := `
		INSERT INTO jobs (
			id, namespace_id, queue_id, workflow_id, workflow_step_id,
			type, payload, status, priority, run_at, idempotency_key,
			max_attempts, attempt_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := s.pool.Exec(ctx, query,
		j.ID, j.NamespaceID, j.QueueID, j.WorkflowID, j.WorkflowStepID,
		j.Type, j.Payload, j.Status, j.Priority, j.RunAt, j.IdempotencyKey,
		j.MaxAttempts, j.AttemptCount, j.CreatedAt, j.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("storage: failed to create job: %w", err)
	}
	return nil
}

// GetJob retrieves a job by its ID.
func (s *Store) GetJob(ctx context.Context, id uid.ID) (*domain.Job, error) {
	query := `
		SELECT id, namespace_id, queue_id, workflow_id, workflow_step_id,
		       type, payload, status, priority, run_at, idempotency_key,
		       max_attempts, attempt_count, locked_by, lease_token, locked_until,
		       last_error, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`
	j := &domain.Job{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&j.ID, &j.NamespaceID, &j.QueueID, &j.WorkflowID, &j.WorkflowStepID,
		&j.Type, &j.Payload, &j.Status, &j.Priority, &j.RunAt, &j.IdempotencyKey,
		&j.MaxAttempts, &j.AttemptCount, &j.LockedBy, &j.LeaseToken, &j.LockedUntil,
		&j.LastError, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("job %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("storage: failed to get job %s: %w", id, err)
	}
	return j, nil
}

// UpdateJobStatus updates the status and updated_at timestamp of a job.
func (s *Store) UpdateJobStatus(ctx context.Context, id uid.ID, status domain.JobStatus) error {
	query := `
		UPDATE jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := s.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("storage: failed to update job status: %w", err)
	}
	return nil
}

// FailJob marks a running job as failed (last attempt exhausted → dead_lettered,
// or still has retries → failed so the reaper can reschedule it).
// It validates the lease token to guard against stale workers.
func (s *Store) FailJob(ctx context.Context, jobID uid.ID, leaseToken uid.ID, errMsg string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Read current attempt info to decide whether to dead-letter.
	var attemptCount, maxAttempts int
	err = tx.QueryRow(ctx,
		`SELECT attempt_count, max_attempts FROM jobs WHERE id = $1 AND lease_token = $2`,
		jobID, leaseToken,
	).Scan(&attemptCount, &maxAttempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaleLease
		}
		return fmt.Errorf("storage: failed to read job for failure: %w", err)
	}

	newStatus := domain.JobStatusRetryScheduled
	if attemptCount >= maxAttempts {
		newStatus = domain.JobStatusDeadLettered
	}

	updateQuery := `
		UPDATE jobs
		SET status = $1,
		    last_error = $2,
		    locked_by = NULL,
		    lease_token = NULL,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE id = $3 AND lease_token = $4
	`
	cmdTag, err := tx.Exec(ctx, updateQuery, newStatus, errMsg, jobID, leaseToken)
	if err != nil {
		return fmt.Errorf("storage: failed to fail job: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrStaleLease
	}

	// Update the attempt record
	_, err = tx.Exec(ctx,
		`UPDATE job_attempts SET status = 'failed', finished_at = NOW(), error = $1
		 WHERE job_id = $2 AND lease_token = $3`,
		errMsg, jobID, leaseToken,
	)
	if err != nil {
		return fmt.Errorf("storage: failed to update attempt record: %w", err)
	}

	return tx.Commit(ctx)
}

// ScheduleRetry transitions a failed job back to 'queued' with a future run_at
// for exponential backoff retries.
func (s *Store) ScheduleRetry(ctx context.Context, jobID uid.ID, leaseToken uid.ID, runAt time.Time, errMsg string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := `
		UPDATE jobs
		SET status = 'queued',
		    run_at = $1,
		    last_error = $2,
		    locked_by = NULL,
		    lease_token = NULL,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE id = $3 AND lease_token = $4 AND status = 'running'
	`
	cmdTag, err := tx.Exec(ctx, updateQuery, runAt, errMsg, jobID, leaseToken)
	if err != nil {
		return fmt.Errorf("storage: failed to schedule retry: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrStaleLease
	}

	_, err = tx.Exec(ctx,
		`UPDATE job_attempts SET status = 'failed', finished_at = NOW(), error = $1
		 WHERE job_id = $2 AND lease_token = $3`,
		errMsg, jobID, leaseToken,
	)
	if err != nil {
		return fmt.Errorf("storage: failed to update attempt on retry: %w", err)
	}

	return tx.Commit(ctx)
}

// --- Timeline ---

// InsertTimelineEvent appends a new event to the timeline log.
func (s *Store) InsertTimelineEvent(ctx context.Context, e *domain.TimelineEvent) error {
	query := `
		INSERT INTO timeline_events (namespace_id, workflow_id, job_id, event_type, message, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := s.pool.QueryRow(ctx, query,
		e.NamespaceID, e.WorkflowID, e.JobID, e.EventType, e.Message, e.Payload, e.CreatedAt,
	).Scan(&e.ID)

	if err != nil {
		return fmt.Errorf("storage: failed to insert timeline event: %w", err)
	}
	return nil
}