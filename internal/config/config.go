package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the TaskForge services.
// Values are loaded from environment variables with sensible defaults.
type Config struct {
	DB     DatabaseConfig
	Redis  RedisConfig
	Server ServerConfig
	Worker WorkerConfig
	Reaper ReaperConfig
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// ServerConfig holds HTTP server parameters.
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// WorkerConfig holds worker runtime parameters.
type WorkerConfig struct {
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
}

// ReaperConfig holds reaper service parameters.
type ReaperConfig struct {
	Interval  time.Duration
	BatchSize int
}

// Load reads configuration from environment variables.
// It uses sensible defaults for local development.
func Load() (*Config, error) {
	cfg := &Config{
		DB: DatabaseConfig{
			Host:     envOrDefault("TASKFORGE_DB_HOST", "localhost"),
			Port:     envIntOrDefault("TASKFORGE_DB_PORT", 5432),
			User:     envOrDefault("TASKFORGE_DB_USER", "taskforge"),
			Password: envOrDefault("TASKFORGE_DB_PASSWORD", "taskforge"),
			DBName:   envOrDefault("TASKFORGE_DB_NAME", "taskforge"),
			SSLMode:  envOrDefault("TASKFORGE_DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     envOrDefault("TASKFORGE_REDIS_ADDR", "localhost:6379"),
			Password: envOrDefault("TASKFORGE_REDIS_PASSWORD", ""),
			DB:       envIntOrDefault("TASKFORGE_REDIS_DB", 0),
		},
		Server: ServerConfig{
			Port:         envIntOrDefault("TASKFORGE_SERVER_PORT", 8080),
			ReadTimeout:  envDurationOrDefault("TASKFORGE_SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: envDurationOrDefault("TASKFORGE_SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Worker: WorkerConfig{
			LeaseDuration:     envDurationOrDefault("TASKFORGE_WORKER_LEASE_DURATION", 60*time.Second),
			HeartbeatInterval: envDurationOrDefault("TASKFORGE_WORKER_HEARTBEAT_INTERVAL", 15*time.Second),
			PollInterval:      envDurationOrDefault("TASKFORGE_WORKER_POLL_INTERVAL", 1*time.Second),
		},
		Reaper: ReaperConfig{
			Interval:  envDurationOrDefault("TASKFORGE_REAPER_INTERVAL", 30*time.Second),
			BatchSize: envIntOrDefault("TASKFORGE_REAPER_BATCH_SIZE", 100),
		},
	}

	return cfg, nil
}

// --- helper functions ---

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}