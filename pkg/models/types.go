package models

import (
	"encoding/json"
	"errors"
	"time"

	"taskforge/internal/domain"
	"taskforge/internal/domain/uid"
)

// CreateNamespaceRequest is the payload for creating a new namespace.
type CreateNamespaceRequest struct {
	Name string `json:"name"`
}

func (req *CreateNamespaceRequest) Validate() error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// CreateQueueRequest is the payload for creating a new queue in a namespace.
type CreateQueueRequest struct {
	Name             string `json:"name"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	RateLimitPerSec  *int   `json:"rate_limit_per_second,omitempty"`
}

func (req *CreateQueueRequest) Validate() error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.ConcurrencyLimit < 0 {
		return errors.New("concurrency_limit cannot be negative")
	}
	return nil
}

// SubmitJobRequest is the payload for enqueueing a new job.
type SubmitJobRequest struct {
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	RunAt          *time.Time      `json:"run_at,omitempty"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	MaxAttempts    *int            `json:"max_attempts,omitempty"`
}

func (req *SubmitJobRequest) Validate() error {
	if req.Type == "" {
		return errors.New("job type is required")
	}
	if req.Payload == nil {
		req.Payload = []byte("{}")
	}
	return nil
}

// JobResponse is the API view of a job.
type JobResponse struct {
	ID             uid.ID           `json:"id"`
	NamespaceID    uid.ID           `json:"namespace_id"`
	QueueID        uid.ID           `json:"queue_id"`
	Type           string           `json:"type"`
	Status         domain.JobStatus `json:"status"`
	Priority       int              `json:"priority"`
	RunAt          time.Time        `json:"run_at"`
	AttemptCount   int              `json:"attempt_count"`
	MaxAttempts    int              `json:"max_attempts"`
	IdempotencyKey *string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

// MapJobToResponse maps a domain.Job to the external JobResponse.
func MapJobToResponse(j *domain.Job) *JobResponse {
	return &JobResponse{
		ID:             j.ID,
		NamespaceID:    j.NamespaceID,
		QueueID:        j.QueueID,
		Type:           j.Type,
		Status:         j.Status,
		Priority:       j.Priority,
		RunAt:          j.RunAt,
		AttemptCount:   j.AttemptCount,
		MaxAttempts:    j.MaxAttempts,
		IdempotencyKey: j.IdempotencyKey,
		CreatedAt:      j.CreatedAt,
	}
}

// APIError is a standard JSON error response.
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewAPIError(err error, msg string) *APIError {
	e := &APIError{Message: msg}
	if err != nil {
		e.Error = err.Error()
	} else {
		e.Error = msg
	}
	return e
}