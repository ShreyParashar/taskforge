package domain

import (
	"time"

	"taskforge/internal/domain/uid"
)

// Namespace represents a tenant, team, or logical environment.
// All queues, jobs, workflows, and idempotency keys are scoped by namespace.
//
// Examples: "default", "security-platform", "payments", "ml-pipelines"
type Namespace struct {
	ID        uid.ID    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Queue groups jobs that should be executed by compatible workers.
// Each queue belongs to a namespace and can have its own concurrency
// and rate limits.
//
// Examples: "default", "security-analysis", "webhooks", "llm-calls"
type Queue struct {
	ID               uid.ID    `json:"id"`
	NamespaceID      uid.ID    `json:"namespace_id"`
	Name             string    `json:"name"`
	ConcurrencyLimit int       `json:"concurrency_limit"`
	RateLimitPerSec  *int      `json:"rate_limit_per_second,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
