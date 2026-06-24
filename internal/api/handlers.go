package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"taskforge/internal/domain"
	"taskforge/internal/domain/uid"
	"taskforge/internal/storage"
	"taskforge/pkg/models"
)

// Server holds the dependencies for the API handlers.
type Server struct {
	store *storage.Store
}

// NewServer creates a new API Server instance.
func NewServer(store *storage.Store) *Server {
	return &Server{store: store}
}

// Healthz returns a simple 200 OK status.
func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{"status": "ok"})
}

// CreateNamespace handles POST /v1/namespaces
func (s *Server) CreateNamespace(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	var req models.CreateNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "invalid json payload"))
		return
	}
	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "validation failed"))
		return
	}

	ns := &domain.Namespace{
		ID:        uid.New(),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateNamespace(r.Context(), ns); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.NewAPIError(err, "failed to create namespace"))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, ns)
}

// CreateQueue handles POST /v1/namespaces/{namespace}/queues
func (s *Server) CreateQueue(w http.ResponseWriter, r *http.Request) {
	namespaceName := chi.URLParam(r, "namespace")
	
	// First, lookup the namespace to ensure it exists and get its ID
	ns, err := s.store.GetNamespaceByName(r.Context(), namespaceName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "namespace not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to look up namespace"))
		}
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	var req models.CreateQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "invalid json payload"))
		return
	}
	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "validation failed"))
		return
	}

	q := &domain.Queue{
		ID:               uid.New(),
		NamespaceID:      ns.ID,
		Name:             req.Name,
		ConcurrencyLimit: req.ConcurrencyLimit,
		RateLimitPerSec:  req.RateLimitPerSec,
		CreatedAt:        time.Now(),
	}

	if err := s.store.CreateQueue(r.Context(), q); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.NewAPIError(err, "failed to create queue"))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, q)
}

// SubmitJob handles POST /v1/namespaces/{namespace}/queues/{queue}/jobs
func (s *Server) SubmitJob(w http.ResponseWriter, r *http.Request) {
	namespaceName := chi.URLParam(r, "namespace")
	queueName := chi.URLParam(r, "queue")

	ns, err := s.store.GetNamespaceByName(r.Context(), namespaceName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "namespace not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to look up namespace"))
		}
		return
	}

	q, err := s.store.GetQueueByName(r.Context(), ns.ID, queueName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "queue not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to look up queue"))
		}
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	var req models.SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "invalid json payload"))
		return
	}
	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "validation failed"))
		return
	}

	job := &domain.Job{
		ID:             uid.New(),
		NamespaceID:    ns.ID,
		QueueID:        q.ID,
		Type:           req.Type,
		Payload:        req.Payload,
		Status:         domain.JobStatusQueued,
		Priority:       req.Priority,
		IdempotencyKey: req.IdempotencyKey,
		AttemptCount:   0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if req.RunAt != nil {
		job.RunAt = *req.RunAt
	} else {
		job.RunAt = time.Now()
	}

	if req.MaxAttempts != nil {
		job.MaxAttempts = *req.MaxAttempts
	} else {
		job.MaxAttempts = domain.DefaultRetryPolicy().MaxAttempts
	}

	if err := s.store.CreateJob(r.Context(), job); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.NewAPIError(err, "failed to submit job"))
		return
	}

	// Create an initial timeline event (non-critical — log failure, don't abort)
	msg := "Job submitted via API"
	if tlErr := s.store.InsertTimelineEvent(r.Context(), &domain.TimelineEvent{
		NamespaceID: ns.ID,
		JobID:       &job.ID,
		EventType:   domain.TimelineJobQueued,
		Message:     &msg,
		Payload:     []byte("{}"),
		CreatedAt:   time.Now(),
	}); tlErr != nil {
		slog.Warn("Failed to insert timeline event", "job_id", job.ID, "error", tlErr)
	}

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, models.MapJobToResponse(job))
}

// GetNamespace handles GET /v1/namespaces/{namespace}
func (s *Server) GetNamespace(w http.ResponseWriter, r *http.Request) {
	namespaceName := chi.URLParam(r, "namespace")
	
	ns, err := s.store.GetNamespaceByName(r.Context(), namespaceName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "namespace not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to get namespace"))
		}
		return
	}
	
	render.Status(r, http.StatusOK)
	render.JSON(w, r, ns)
}

// GetQueue handles GET /v1/namespaces/{namespace}/queues/{queue}
func (s *Server) GetQueue(w http.ResponseWriter, r *http.Request) {
	namespaceName := chi.URLParam(r, "namespace")
	queueName := chi.URLParam(r, "queue")

	ns, err := s.store.GetNamespaceByName(r.Context(), namespaceName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "namespace not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to look up namespace"))
		}
		return
	}

	q, err := s.store.GetQueueByName(r.Context(), ns.ID, queueName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "queue not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to get queue"))
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, q)
}

// GetJob handles GET /v1/namespaces/{namespace}/queues/{queue}/jobs/{job_id}
func (s *Server) GetJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "job_id")
	
	jobID, err := uid.Parse(jobIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.NewAPIError(err, "invalid job_id format"))
		return
	}

	job, err := s.store.GetJob(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, models.NewAPIError(err, "job not found"))
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, models.NewAPIError(err, "failed to get job"))
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, models.MapJobToResponse(job))
}