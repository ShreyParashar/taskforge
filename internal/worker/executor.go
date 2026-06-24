package worker

import (
	"context"
	"log/slog"
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
// It blocks if the executor has reached its concurrency limit.
func (e *Executor) ExecuteJob(ctx context.Context, job *domain.Job) {
	// Acquire a semaphore token. This blocks if we're at the concurrency limit.
	e.sem <- struct{}{}

	go func() {
		defer func() {
			// Release the semaphore token when done
			<-e.sem
			
			// Basic panic recovery for safety
			if r := recover(); r != nil {
				slog.Error("Panic during job execution", "job_id", job.ID, "panic", r)
				// In a real system, we'd log this and potentially mark the job as failed here.
			}
		}()

		start := time.Now()

		// 1. Find the handler for this job type
		handler, err := e.registry.Get(job.Type)
		if err != nil {
			slog.Error("No handler found for job", "job_id", job.ID, "type", job.Type)
			// TODO: Phase 5 - fail the job in DB
			return
		}

		// 2. Execute the business logic
		execErr := handler.Execute(ctx, job)
		
		durationMs := int(time.Since(start).Milliseconds())

		// 3. Handle the result
		if execErr != nil {
			slog.Error("Job failed", "job_id", job.ID, "error", execErr)
			// TODO: Phase 5 - Handle retries or transition to dead_lettered
			return
		}

		// 4. On success, mark it complete in the database
		err = e.store.CompleteJob(context.Background(), job.ID, *job.LeaseToken, durationMs)
		if err != nil {
			slog.Error("Failed to mark job complete", "job_id", job.ID, "error", err)
			return
		}

		// Create timeline event for completion
		msg := "Job completed successfully"
		_ = e.store.InsertTimelineEvent(context.Background(), &domain.TimelineEvent{
			NamespaceID: job.NamespaceID,
			JobID:       &job.ID,
			EventType:   domain.TimelineJobSucceeded,
			Message:     &msg,
			CreatedAt:   time.Now(),
		})

		slog.Info("Job completed successfully", "job_id", job.ID, "duration_ms", durationMs)
	}()
}