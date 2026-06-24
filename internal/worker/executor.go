package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"taskforge/internal/domain"
	"taskforge/internal/storage"
	"taskforge/pkg/handlers"
)

// Executor manages concurrent execution of jobs, bounded by a concurrency limit.
type Executor struct {
	store    *storage.Store
	registry *handlers.Registry
	sem      chan struct{} // Semaphore for concurrency control
	wg       sync.WaitGroup // Tracks all in-flight goroutines for graceful drain
}

// NewExecutor creates a new job executor.
func NewExecutor(store *storage.Store, registry *handlers.Registry, concurrencyLimit int) *Executor {
	return &Executor{
		store:    store,
		registry: registry,
		sem:      make(chan struct{}, concurrencyLimit),
	}
}

// ExecuteJob runs the job in a bounded goroutine.
// It blocks if the executor has reached its concurrency limit, providing
// natural back-pressure to the Leaser so it does not over-lease from the DB.
func (e *Executor) ExecuteJob(ctx context.Context, job *domain.Job) {
	// Acquire a semaphore token. This blocks if we're at the concurrency limit.
	// Use a select so that shutdown signals are still respected while we wait.
	select {
	case e.sem <- struct{}{}:
		// Acquired
	case <-ctx.Done():
		slog.Warn("Executor context cancelled while waiting for semaphore, dropping job", "job_id", job.ID)
		return
	}

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() { <-e.sem }()

		// Catch panics so a bad handler cannot crash the entire worker process.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic recovered during job execution", "job_id", job.ID, "panic", r)
				e.failOrRetry(job, "worker panic during execution")
			}
		}()

		e.runJob(ctx, job)
	}()
}

// runJob contains the core execution logic for a single job.
func (e *Executor) runJob(ctx context.Context, job *domain.Job) {
	// Use a short-lived context for the handler so that the shutdown signal
	// propagates to the handler, but we still have time to write the result to DB.
	handler, err := e.registry.Get(job.Type)
	if err != nil {
		slog.Error("No handler registered for job type", "job_id", job.ID, "type", job.Type)
		e.failOrRetry(job, "no handler registered for job type: "+job.Type)
		return
	}

	start := time.Now()
	execErr := handler.Execute(ctx, job)
	durationMs := int(time.Since(start).Milliseconds())

	if execErr != nil {
		slog.Error("Job handler returned error", "job_id", job.ID, "error", execErr)
		e.failOrRetry(job, execErr.Error())
		return
	}

	// Use a fresh context for the DB write — the shutdown context may already be cancelled,
	// but we always want to mark a successfully-completed job as done.
	writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.store.CompleteJob(writeCtx, job.ID, *job.LeaseToken, durationMs); err != nil {
		if errors.Is(err, storage.ErrStaleLease) {
			slog.Warn("Stale lease on complete — another worker may have already completed this job", "job_id", job.ID)
			return
		}
		slog.Error("Failed to mark job complete in DB", "job_id", job.ID, "error", err)
		return
	}

	// Non-critical: record success in timeline.
	msg := "Job completed successfully"
	if tlErr := e.store.InsertTimelineEvent(writeCtx, &domain.TimelineEvent{
		NamespaceID: job.NamespaceID,
		JobID:       &job.ID,
		EventType:   domain.TimelineJobSucceeded,
		Message:     &msg,
		CreatedAt:   time.Now(),
	}); tlErr != nil {
		slog.Warn("Failed to insert success timeline event", "job_id", job.ID, "error", tlErr)
	}

	slog.Info("Job completed successfully", "job_id", job.ID, "duration_ms", durationMs)
}

// failOrRetry decides whether to retry a failed job or dead-letter it,
// based on the domain retry policy.
func (e *Executor) failOrRetry(job *domain.Job, errMsg string) {
	if job.LeaseToken == nil {
		slog.Error("Cannot fail job — lease token is nil", "job_id", job.ID)
		return
	}

	writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	retryPolicy := job.RetryPolicy()
	if retryPolicy.ShouldRetry(job.AttemptCount) {
		nextRunAt := retryPolicy.NextRetryAt(job.AttemptCount)
		slog.Info("Scheduling job for retry", "job_id", job.ID, "attempt", job.AttemptCount, "retry_at", nextRunAt)

		if err := e.store.ScheduleRetry(writeCtx, job.ID, *job.LeaseToken, nextRunAt, errMsg); err != nil {
			slog.Error("Failed to schedule job retry", "job_id", job.ID, "error", err)
		}

		// Non-critical: timeline event
		msg := "Job scheduled for retry"
		_ = e.store.InsertTimelineEvent(writeCtx, &domain.TimelineEvent{
			NamespaceID: job.NamespaceID,
			JobID:       &job.ID,
			EventType:   domain.TimelineJobRetryScheduled,
			Message:     &msg,
			CreatedAt:   time.Now(),
		})
		return
	}

	slog.Warn("Job exhausted all retries, dead-lettering", "job_id", job.ID, "attempts", job.AttemptCount)
	if err := e.store.FailJob(writeCtx, job.ID, *job.LeaseToken, errMsg); err != nil {
		slog.Error("Failed to dead-letter job", "job_id", job.ID, "error", err)
	}

	// Non-critical: timeline event
	msg := "Job dead-lettered after exhausting retries"
	_ = e.store.InsertTimelineEvent(writeCtx, &domain.TimelineEvent{
		NamespaceID: job.NamespaceID,
		JobID:       &job.ID,
		EventType:   domain.TimelineJobDeadLettered,
		Message:     &msg,
		CreatedAt:   time.Now(),
	})
}

// Drain waits until all currently in-flight jobs have finished executing.
// Should be called after cancelling the leaser context on shutdown.
func (e *Executor) Drain() {
	slog.Info("Draining in-flight jobs...")
	e.wg.Wait()
	slog.Info("All in-flight jobs drained.")
}