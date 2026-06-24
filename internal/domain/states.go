package domain

import "fmt"

// --- Job States ---

// JobStatus represents the current state of a job in the state machine.
type JobStatus string

const (
	JobStatusQueued         JobStatus = "queued"
	JobStatusRunning        JobStatus = "running"
	JobStatusRetryScheduled JobStatus = "retry_scheduled"
	JobStatusSucceeded      JobStatus = "succeeded"
	JobStatusDeadLettered   JobStatus = "dead_lettered"
	JobStatusCancelled      JobStatus = "cancelled"
)

// validJobTransitions defines the allowed state machine transitions for jobs.
//
// State machine:
//
//	queued          -> running, cancelled
//	running         -> succeeded, retry_scheduled, dead_lettered, cancelled
//	retry_scheduled -> queued, cancelled
//	succeeded       -> (terminal)
//	dead_lettered   -> (terminal, unless manual retry)
//	cancelled       -> (terminal)
var validJobTransitions = map[JobStatus][]JobStatus{
	JobStatusQueued:         {JobStatusRunning, JobStatusCancelled},
	JobStatusRunning:        {JobStatusSucceeded, JobStatusRetryScheduled, JobStatusDeadLettered, JobStatusCancelled},
	JobStatusRetryScheduled: {JobStatusQueued, JobStatusCancelled},
	// Terminal states have no outgoing transitions.
	JobStatusSucceeded:    {},
	JobStatusDeadLettered: {},
	JobStatusCancelled:    {},
}

// ValidateJobTransition checks whether transitioning from `from` to `to` is allowed.
// Returns an error if the transition is invalid.
func ValidateJobTransition(from, to JobStatus) error {
	allowed, exists := validJobTransitions[from]
	if !exists {
		return fmt.Errorf("unknown job status: %q", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid job transition: %q -> %q", from, to)
}

// IsTerminalJobStatus returns true if the job status is a terminal state
// (no further transitions are possible without manual intervention).
func IsTerminalJobStatus(s JobStatus) bool {
	return s == JobStatusSucceeded || s == JobStatusDeadLettered || s == JobStatusCancelled
}

// --- Workflow States ---

// WorkflowStatus represents the current state of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusSucceeded WorkflowStatus = "succeeded"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// --- Workflow Step States ---

// WorkflowStepStatus represents the current state of a workflow step.
type WorkflowStepStatus string

const (
	WorkflowStepStatusWaiting   WorkflowStepStatus = "waiting"
	WorkflowStepStatusQueued    WorkflowStepStatus = "queued"
	WorkflowStepStatusRunning   WorkflowStepStatus = "running"
	WorkflowStepStatusSucceeded WorkflowStepStatus = "succeeded"
	WorkflowStepStatusFailed    WorkflowStepStatus = "failed"
	WorkflowStepStatusCancelled WorkflowStepStatus = "cancelled"
)
