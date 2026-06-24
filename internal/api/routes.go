package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"taskforge/internal/storage"
)

// NewRouter initializes a chi router with middleware and registers all API endpoints.
func NewRouter(store *storage.Store) *chi.Mux {
	r := chi.NewRouter()

	// 1. Core Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger()) // Our custom slog middleware
	r.Use(middleware.Recoverer)
	
	server := NewServer(store)

	// 2. Health & Diagnostics
	r.Get("/healthz", server.Healthz)

	// 3. API V1 Routes
	r.Route("/v1", func(r chi.Router) {
		// Namespaces
		r.Post("/namespaces", server.CreateNamespace)
		r.Get("/namespaces/{namespace}", server.GetNamespace)
		
		// Queues
		r.Post("/namespaces/{namespace}/queues", server.CreateQueue)
		r.Get("/namespaces/{namespace}/queues/{queue}", server.GetQueue)
		
		// Jobs
		r.Post("/namespaces/{namespace}/queues/{queue}/jobs", server.SubmitJob)
		r.Get("/namespaces/{namespace}/queues/{queue}/jobs/{job_id}", server.GetJob)
	})

	return r
}