package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"taskforge/internal/domain/uid"
	"taskforge/internal/storage"
)

// Leaser continuously polls a specific queue for jobs and feeds them to the executor.
type Leaser struct {
	store         *storage.Store
	workerID      string
	queueID       uid.ID
	leaseDuration time.Duration
	pollInterval  time.Duration
	executor      *Executor
}

// NewLeaser creates a new job leaser for a specific queue.
func NewLeaser(store *storage.Store, workerID string, queueID uid.ID, executor *Executor) *Leaser {
	return &Leaser{
		store:         store,
		workerID:      workerID,
		queueID:       queueID,
		leaseDuration: 5 * time.Minute,
		pollInterval:  500 * time.Millisecond,
		executor:      executor,
	}
}

// Start begins the continuous polling loop. It blocks until the context is canceled.
func (l *Leaser) Start(ctx context.Context) {
	slog.Info("Starting leaser for queue", "queue_id", l.queueID, "worker_id", l.workerID)
	
	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping leaser", "queue_id", l.queueID)
			return
		default:
			// Attempt to lease a job
			job, err := l.store.LeaseJob(ctx, l.queueID, l.workerID, l.leaseDuration)
			if err != nil {
				if errors.Is(err, storage.ErrNoJobsAvailable) {
					// No jobs available, wait for the next tick
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						continue
					}
				}
				slog.Error("Error leasing job", "error", err, "queue_id", l.queueID)
				time.Sleep(1 * time.Second) // backoff on error
				continue
			}

			slog.Info("Successfully leased job", "job_id", job.ID, "type", job.Type)

			// We have a job! Hand it off to the executor.
			// The executor manages its own concurrency limits and will block
			// here if the worker is already full.
			l.executor.ExecuteJob(ctx, job)
		}
	}
}