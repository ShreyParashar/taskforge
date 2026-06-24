package handlers

import (
	"context"
	"log/slog"

	"taskforge/internal/domain"
)

// EchoHandler is a simple job handler that logs the job's payload to the console.
// It is used for end-to-end testing of the pipeline.
type EchoHandler struct{}

// Execute logs the payload of the job.
func (h *EchoHandler) Execute(ctx context.Context, job *domain.Job) error {
	slog.Info("EchoHandler executing job",
		slog.String("job_id", string(job.ID)),
		slog.String("type", job.Type),
		slog.String("payload", string(job.Payload)),
	)
	
	// Simulating some short processing time
	// time.Sleep(100 * time.Millisecond)

	return nil
}