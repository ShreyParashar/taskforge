package domain

import (
	"encoding/json"
	"time"

	"taskforge/internal/domain/uid"
)

// Job represents a single durable unit of work in the system.
// A job belongs to a namespace and queue, may be part of a workflow,
// and tracks its own retry/lease state.
type Job struct {
	ID          uid.ID  `json:"id"`
	NamespaceID uid.ID  `json:"namespace_id"`
	QueueID     uid.ID  `json:"queue_id"`
	WorkflowID  *uid.ID `json:"workflow_id,omitempty"`
	WorkflowStepID *uid.ID `json:"workflow_step_id,omitempty"`

	// Job definition
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Status  JobStatus       `json:"status"`

	// Scheduling and priority
	Priority int       `json:"priority"`
	RunAt    time.Time `json:"run_at"`

	// Idempotency
	IdempotencyKey *string `json:"idempotency_key,omitempty"`

	// Retry state
	MaxAttempts  int `json:"max_attempts"`
	AttemptCount int `json:"attempt_count"`

	// Lease state — populated when a worker holds this job
	LockedBy    *string  `json:"locked_by,omitempty"`
	LeaseToken  *uid.ID  `json:"lease_token,omitempty"`
	LockedUntil *time.Time `json:"locked_until,omitempty"`

	// Error tracking
	LastError *string `json:"last_error,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobAttempt records a single execution attempt of a job.
// Each time a worker leases and runs a job, an attempt row is created.
// This provides a full audit trail: which worker ran it, how long it took,
// and whether it succeeded or failed.
type JobAttempt struct {
	ID         uid.ID    `json:"id"`
	JobID      uid.ID    `json:"job_id"`
	AttemptNo  int       `json:"attempt_no"`
	WorkerID   string    `json:"worker_id"`
	LeaseToken uid.ID    `json:"lease_token"`
	Status     JobStatus `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      *string   `json:"error,omitempty"`
	DurationMs *int      `json:"duration_ms,omitempty"`
}

// RetryPolicy configures how a job should be retried on failure.
type RetryPolicy struct {
	MaxAttempts  int           `json:"max_attempts"`
	BaseBackoff  time.Duration `json:"base_backoff"`
	MaxBackoff   time.Duration `json:"max_backoff"`
	JitterFactor float64       `json:"jitter_factor"` // 0.0 to 1.0, typically 0.2
}

// DefaultRetryPolicy returns a sensible default retry configuration.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:  3,
		BaseBackoff:  2 * time.Second,
		MaxBackoff:   5 * time.Minute,
		JitterFactor: 0.2,
	}
}

// ShouldRetry determines whether a failed job should be retried
// based on the current attempt count and the retry policy.
func (p RetryPolicy) ShouldRetry(attemptCount int) bool {
	return attemptCount < p.MaxAttempts
}
