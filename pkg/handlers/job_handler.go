package handlers

import (
	"context"
	"errors"
	"sync"

	"taskforge/internal/domain"
)

// JobHandler is the interface that all background job processors must implement.
type JobHandler interface {
	// Execute performs the actual work of the job.
	// If it returns an error, the job will be retried according to its retry policy.
	Execute(ctx context.Context, job *domain.Job) error
}

// Registry manages the mapping between a job's string type and its handler logic.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

// NewRegistry creates a new empty job handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]JobHandler),
	}
}

// Register registers a handler for a specific job type.
func (r *Registry) Register(jobType string, handler JobHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = handler
}

// Get returns the handler for a given job type, or an error if not found.
func (r *Registry) Get(jobType string) (JobHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[jobType]
	if !exists {
		return nil, errors.New("no handler registered for job type: " + jobType)
	}
	return handler, nil
}