package domain

import (
	"encoding/json"
	"time"

	"taskforge/internal/domain/uid"
)

// TimelineEventType constants define all possible timeline event types.
// The timeline is an append-only audit log for jobs and workflows,
// enabling operators to inspect exactly what happened during execution.
type TimelineEventType string

const (
	// Workflow events
	TimelineWorkflowStarted   TimelineEventType = "workflow_started"
	TimelineWorkflowSucceeded TimelineEventType = "workflow_succeeded"
	TimelineWorkflowFailed    TimelineEventType = "workflow_failed"
	TimelineWorkflowCancelled TimelineEventType = "workflow_cancelled"

	// Workflow step events
	TimelineStepQueued    TimelineEventType = "step_queued"
	TimelineStepStarted   TimelineEventType = "step_started"
	TimelineStepSucceeded TimelineEventType = "step_succeeded"
	TimelineStepFailed    TimelineEventType = "step_failed"

	// Job events
	TimelineJobQueued         TimelineEventType = "job_queued"
	TimelineJobLeased         TimelineEventType = "job_leased"
	TimelineJobSucceeded      TimelineEventType = "job_succeeded"
	TimelineJobFailed         TimelineEventType = "job_failed"
	TimelineJobRetryScheduled TimelineEventType = "job_retry_scheduled"
	TimelineJobDeadLettered   TimelineEventType = "job_dead_lettered"
	TimelineJobCancelled      TimelineEventType = "job_cancelled"

	// Worker events
	TimelineWorkerHeartbeat TimelineEventType = "worker_heartbeat"
	TimelineLeaseExpired    TimelineEventType = "lease_expired"
	TimelineLeaseRecovered  TimelineEventType = "lease_recovered"

	// Rate limiting / circuit breaker events
	TimelineRateLimited    TimelineEventType = "rate_limited"
	TimelineCircuitOpened  TimelineEventType = "circuit_opened"
	TimelineCircuitClosed  TimelineEventType = "circuit_closed"
)

// TimelineEvent is a single entry in the append-only timeline log.
// Events reference a namespace and optionally a workflow and/or job.
type TimelineEvent struct {
	ID          int64             `json:"id"`
	NamespaceID uid.ID           `json:"namespace_id"`
	WorkflowID  *uid.ID          `json:"workflow_id,omitempty"`
	JobID       *uid.ID          `json:"job_id,omitempty"`
	EventType   TimelineEventType `json:"event_type"`
	Message     *string          `json:"message,omitempty"`
	Payload     json.RawMessage  `json:"payload"`
	CreatedAt   time.Time        `json:"created_at"`
}

// Schedule represents a recurring cron-based job schedule.
type Schedule struct {
	ID              uid.ID          `json:"id"`
	NamespaceID     uid.ID          `json:"namespace_id"`
	QueueID         uid.ID          `json:"queue_id"`
	Name            string          `json:"name"`
	CronExpr        string          `json:"cron_expr"`
	JobType         string          `json:"job_type"`
	PayloadTemplate json.RawMessage `json:"payload_template"`
	Enabled         bool            `json:"enabled"`
	NextRunAt       time.Time       `json:"next_run_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}