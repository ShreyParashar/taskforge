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

var (
	// ErrNoJobsAvailable is returned when a worker tries to lease a job but none are ready.
	ErrNoJobsAvailable = errors.New("no jobs available")
	// ErrStaleLease is returned when a worker tries to complete a job but its lease has expired.
	ErrStaleLease = errors.New("stale lease token")
)

// LeaseJob attempts to find and lock a ready job from the specified queue.
// It uses FOR UPDATE SKIP LOCKED to allow many workers to scan the queue concurrently
// without blocking each other.
func (s *Store) LeaseJob(ctx context.Context, queueID uid.ID, workerID string, leaseDuration time.Duration) (*domain.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Find the highest priority job that is ready to run
	findQuery := `
		SELECT id
		FROM jobs
		WHERE status = 'queued'
		  AND run_at <= NOW()
		  AND queue_id = $1
		ORDER BY priority DESC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`
	var jobID uid.ID
	err = tx.QueryRow(ctx, findQuery, queueID).Scan(&jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoJobsAvailable
		}
		return nil, fmt.Errorf("failed to find job to lease: %w", err)
	}

	// We found a job and locked it. Now transition it to running.
	leaseToken := uid.New()
	lockedUntil := time.Now().Add(leaseDuration)

	updateQuery := `
		UPDATE jobs
		SET status = 'running',
		    locked_by = $1,
		    lease_token = $2,
		    locked_until = $3,
		    attempt_count = attempt_count + 1,
		    updated_at = NOW()
		WHERE id = $4
		RETURNING type, payload, status, priority, run_at, idempotency_key, 
		          max_attempts, attempt_count, created_at, updated_at, namespace_id, workflow_id, workflow_step_id
	`
	
	job := &domain.Job{
		ID:          jobID,
		QueueID:     queueID,
		LockedBy:    &workerID,
		LeaseToken:  &leaseToken,
		LockedUntil: &lockedUntil,
	}

	err = tx.QueryRow(ctx, updateQuery, workerID, leaseToken, lockedUntil, jobID).Scan(
		&job.Type, &job.Payload, &job.Status, &job.Priority, &job.RunAt, &job.IdempotencyKey,
		&job.MaxAttempts, &job.AttemptCount, &job.CreatedAt, &job.UpdatedAt,
		&job.NamespaceID, &job.WorkflowID, &job.WorkflowStepID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update leased job: %w", err)
	}

	// Insert the job attempt record for auditability
	attemptQuery := `
		INSERT INTO job_attempts (id, job_id, attempt_no, worker_id, lease_token, status, started_at)
		VALUES ($1, $2, $3, $4, $5, 'running', NOW())
	`
	attemptID := uid.New()
	_, err = tx.Exec(ctx, attemptQuery, attemptID, jobID, job.AttemptCount, workerID, leaseToken)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit lease transaction: %w", err)
	}

	return job, nil
}

// CompleteJob marks a running job as successfully completed.
// It requires the correct leaseToken to ensure a stale worker cannot
// complete a job that was already recovered by the reaper.
func (s *Store) CompleteJob(ctx context.Context, jobID uid.ID, leaseToken uid.ID, durationMs int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Only update if the job is still running and the lease token matches
	updateJobQuery := `
		UPDATE jobs
		SET status = 'succeeded',
		    locked_by = NULL,
		    lease_token = NULL,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'running' AND lease_token = $2
	`
	cmdTag, err := tx.Exec(ctx, updateJobQuery, jobID, leaseToken)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	
	if cmdTag.RowsAffected() == 0 {
		return ErrStaleLease
	}

	// Update the attempt record
	updateAttemptQuery := `
		UPDATE job_attempts
		SET status = 'succeeded',
		    finished_at = NOW(),
		    duration_ms = $1
		WHERE job_id = $2 AND lease_token = $3
	`
	_, err = tx.Exec(ctx, updateAttemptQuery, durationMs, jobID, leaseToken)
	if err != nil {
		return fmt.Errorf("failed to update job attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit complete transaction: %w", err)
	}

	return nil
}
