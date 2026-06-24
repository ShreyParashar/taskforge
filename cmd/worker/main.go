package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskforge/internal/config"
	"taskforge/internal/domain/uid"
	"taskforge/internal/storage"
	"taskforge/internal/worker"
	"taskforge/pkg/handlers"
)

func main() {
	// 1. Initialize Structured Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Println("===========================================")
	fmt.Println("  TaskForge — Worker Pool")
	fmt.Println("===========================================")
	fmt.Printf("  Database:      %s:%d/%s\n", cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName)
	fmt.Printf("  Concurrency:   %d\n", cfg.Worker.ConcurrencyLimit)
	fmt.Println("===========================================")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())

	// 3. Initialize Storage Layer
	store, err := storage.NewStore(ctx, cfg.DB)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// 4. Initialize Handler Registry
	registry := handlers.NewRegistry()
	
	// Register the dummy echo handler for testing
	registry.Register("echo", &handlers.EchoHandler{})

	// 5. Initialize the Worker Engine
	// For testing, we just generate a random UUID for the queue we want to listen to.
	// In reality, the worker would look up the queue ID by name or be configured with it.
	// We'll pass a dummy queue ID for now, which we can override in tests.
	testQueueID := uid.ID("queue-12345") // This will need to be a real queue ID in production
	workerID := fmt.Sprintf("worker-%d", time.Now().Unix())

	executor := worker.NewExecutor(store, registry, cfg.Worker.ConcurrencyLimit)
	leaser := worker.NewLeaser(store, workerID, testQueueID, executor)

	// 6. Graceful Shutdown & Run
	go func() {
		slog.Info("Starting Worker Leaser", "worker_id", workerID)
		leaser.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	<-quit
	slog.Info("Worker is shutting down gracefully...")
	
	// Stop the leaser from picking up new jobs
	cancel()

	// Give the executor time to finish running active jobs (drain)
	slog.Info("Waiting for active jobs to finish (drain)...")
	time.Sleep(2 * time.Second) // In a real system, we'd use a WaitGroup in the executor

	slog.Info("Worker exited gracefully")
}