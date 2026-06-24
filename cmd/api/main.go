package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskforge/internal/api"
	"taskforge/internal/config"
	"taskforge/internal/storage"
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
	fmt.Println("  TaskForge — Workflow Orchestration Engine")
	fmt.Println("===========================================")
	fmt.Printf("  Server port:   %d\n", cfg.Server.Port)
	fmt.Printf("  Database:      %s:%d/%s\n", cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName)
	fmt.Printf("  Redis:         %s\n", cfg.Redis.Addr)
	fmt.Println("===========================================")
	fmt.Println()

	ctx := context.Background()

	// 3. Run Database Migrations
	slog.Info("Applying database migrations...")
	if err := storage.MigrateUp(cfg.DB, "migrations"); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("Database migrations applied successfully")

	// 4. Initialize Storage Layer (PostgreSQL)
	store, err := storage.NewStore(ctx, cfg.DB)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("Connected to PostgreSQL successfully")

	// 5. Setup HTTP Router & Server
	router := api.NewRouter(store)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 6. Graceful Shutdown
	go func() {
		slog.Info("Starting TaskForge API Server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	// Accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	<-quit
	slog.Info("Server is shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited gracefully")
}
