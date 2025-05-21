package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/BogdanDolia/ops-butler/internal/scheduler"
	"github.com/BogdanDolia/ops-butler/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := scheduler.NewConfig()

	// Initialize logger
	l, err := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Sync()

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnvAsInt("DB_PORT", 5432),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "k8s_ops_portal"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		l.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Create scheduler
	s, err := scheduler.NewScheduler(cfg, l, db)
	if err != nil {
		l.Fatal("Failed to create scheduler", zap.Error(err))
		os.Exit(1)
	}

	// Start scheduler
	if err := s.Start(); err != nil {
		l.Fatal("Failed to start scheduler", zap.Error(err))
		os.Exit(1)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Stop scheduler
	if err := s.Stop(); err != nil {
		l.Error("Failed to stop scheduler", zap.Error(err))
		os.Exit(1)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
