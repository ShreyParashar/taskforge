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

	// The queue ID to listen on is required via environment variable.
	// In production, this would typically be discovered via a queue name → ID lookup.
	queueIDStr := os.Getenv("TASKFORGE_WORKER_QUEUE_ID")
	if queueIDStr == "" {
		log.Fatal("TASKFORGE_WORKER_QUEUE_ID environment variable is required")
	}
	queueID := uid.ID(queueIDStr)

	workerID := os.Getenv("TASKFORGE_WORKER_ID")
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}

	fmt.Println("===========================================")
	fmt.Println("  TaskForge — Worker Pool")
	fmt.Println("===========================================")
	fmt.Printf("  Database:      %s:%d/%s\n", cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName)
	fmt.Printf("  Worker ID:     %s\n", workerID)
	fmt.Printf("  Queue ID:      %s\n", queueID)
	fmt.Printf("  Concurrency:   %d\n", cfg.Worker.ConcurrencyLimit)
	fmt.Printf("  Poll interval: %s\n", cfg.Worker.PollInterval)
	fmt.Printf("  Lease duration:%s\n", cfg.Worker.LeaseDuration)
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

	// Register all job handlers here
	registry.Register("echo", &handlers.EchoHandler{})
	// registry.Register("send_email", &yourpackage.SendEmailHandler{})

	// 5. Initialize the Worker Engine
	executor := worker.NewExecutor(store, registry, cfg.Worker.ConcurrencyLimit)
	leaser := worker.NewLeaser(store, workerID, queueID, executor, worker.LeaserConfig{
		LeaseDuration: cfg.Worker.LeaseDuration,
		PollInterval:  cfg.Worker.PollInterval,
	})

	// 6. Start the leaser in a background goroutine
	go func() {
		slog.Info("Starting Worker Leaser", "worker_id", workerID, "queue_id", queueID)
		leaser.Start(ctx)
	}()

	// 7. Graceful Shutdown — wait for OS signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutdown signal received — stopping leaser and draining active jobs...")

	// Step 1: Cancel the context — stops the leaser from picking up new jobs
	cancel()

	// Step 2: Wait for all in-flight goroutines to finish (WaitGroup drain)
	// The executor uses context.Background() for DB writes, so they will complete
	// even after cancel() is called.
	executor.Drain()

	// store.Close() is called via defer after Drain() returns — safe ordering.
	slog.Info("Worker exited gracefully")
}