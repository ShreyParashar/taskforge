package domain

import (
	"time"

	"taskforge/internal/domain/uid"
)

// IdempotencyRecord tracks idempotency keys to prevent duplicate
// job or workflow creation. When a client submits a request with an
// idempotency key, the system checks for an existing record.
//
// If a record exists with the same key but different request hash,
// a 409 Conflict is returned. If the record matches, the original
// resource is returned without creating a duplicate.
type IdempotencyRecord struct {
	ID           uid.ID    `json:"id"`
	NamespaceID  uid.ID    `json:"namespace_id"`
	Key          string    `json:"key"`
	RequestHash  string    `json:"request_hash"`
	ResourceType string    `json:"resource_type"` // "job" or "workflow"
	ResourceID   uid.ID    `json:"resource_id"`
	CreatedAt    time.Time `json:"created_at"`
}