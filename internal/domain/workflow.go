package domain

import (
	"encoding/json"
	"time"

	"taskforge/internal/domain/uid"
)

// Workflow represents a group of dependent steps that form a DAG.
// Each step creates a job, and the workflow engine advances
// through the DAG as jobs complete.
type Workflow struct {
	ID             uid.ID          `json:"id"`
	NamespaceID    uid.ID          `json:"namespace_id"`
	Name           string          `json:"name"`
	Status         WorkflowStatus  `json:"status"`
	Input          json.RawMessage `json:"input"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// WorkflowStep represents a single step in a workflow DAG.
// Each step is backed by a job. Steps declare their dependencies
// via DependsOn, which references other step names in the same workflow.
type WorkflowStep struct {
	ID         uid.ID             `json:"id"`
	WorkflowID uid.ID            `json:"workflow_id"`
	Name       string             `json:"name"`
	JobType    string             `json:"job_type"`
	QueueName  string             `json:"queue_name"`
	Payload    json.RawMessage    `json:"payload"`
	DependsOn  []string           `json:"depends_on"`
	Status     WorkflowStepStatus `json:"status"`
	JobID      *uid.ID            `json:"job_id,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// WorkflowDefinition is the input format for creating a workflow.
// It contains the workflow metadata and the list of step definitions.
type WorkflowDefinition struct {
	Name           string                   `json:"name"`
	IdempotencyKey *string                  `json:"idempotency_key,omitempty"`
	Input          json.RawMessage          `json:"input"`
	Steps          []WorkflowStepDefinition `json:"steps"`
}

// WorkflowStepDefinition is the input format for defining a workflow step.
type WorkflowStepDefinition struct {
	Name      string          `json:"name"`
	Queue     string          `json:"queue"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	DependsOn []string        `json:"depends_on"`
}