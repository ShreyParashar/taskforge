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

// LeaserConfig holds the configuration for the leaser.
type LeaserConfig struct {
	LeaseDuration time.Duration
	PollInterval  time.Duration
}

// NewLeaser creates a new job leaser for a specific queue.
func NewLeaser(store *storage.Store, workerID string, queueID uid.ID, executor *Executor, cfg LeaserConfig) *Leaser {
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = 5 * time.Minute
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	return &Leaser{
		store:         store,
		workerID:      workerID,
		queueID:       queueID,
		leaseDuration: cfg.LeaseDuration,
		pollInterval:  cfg.PollInterval,
		executor:      executor,
	}
}

// Start begins the continuous polling loop. It blocks until the context is canceled.
// When jobs are available it leases them as fast as the executor can absorb them
// (the executor semaphore provides back-pressure). When the queue is empty, it
// waits one full poll interval before trying again.
func (l *Leaser) Start(ctx context.Context) {
	slog.Info("Starting leaser for queue", "queue_id", l.queueID, "worker_id", l.workerID)

	for {
		// Check if we've been asked to stop before attempting a lease.
		select {
		case <-ctx.Done():
			slog.Info("Stopping leaser", "queue_id", l.queueID)
			return
		default:
		}

		job, err := l.store.LeaseJob(ctx, l.queueID, l.workerID, l.leaseDuration)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				slog.Info("Leaser context cancelled, stopping")
				return
			}
			if errors.Is(err, storage.ErrNoJobsAvailable) {
				// Queue is empty — wait a full poll interval before trying again.
				slog.Debug("No jobs available, waiting", "poll_interval", l.pollInterval)
				select {
				case <-ctx.Done():
					slog.Info("Stopping leaser", "queue_id", l.queueID)
					return
				case <-time.After(l.pollInterval):
					continue
				}
			}
			// Unexpected DB error — back off before retrying.
			slog.Error("Error leasing job", "error", err, "queue_id", l.queueID)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		slog.Info("Successfully leased job", "job_id", job.ID, "type", job.Type)

		// Hand off to the executor. This BLOCKS if the worker has reached its
		// concurrency limit (semaphore is full) — providing natural back-pressure
		// so we do not over-lease from the DB.
		l.executor.ExecuteJob(ctx, job)
	}
}