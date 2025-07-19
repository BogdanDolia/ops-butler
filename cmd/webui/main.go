package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/webui"
	"github.com/BogdanDolia/ops-butler/new-ops-butler/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", "json", "Log format (json, console)")
	flag.Parse()

	// Initialize logger
	l, err := logger.NewLogger(&logger.Config{
		Level:  *logLevel,
		Format: *logFormat,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer l.Sync()

	// Load configuration
	cfg := &webui.Config{}
	if *configFile != "" {
		// Load from file
		data, err := os.ReadFile(*configFile)
		if err != nil {
			l.Fatal("Failed to read configuration file", zap.Error(err))
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			l.Fatal("Failed to parse configuration file", zap.Error(err))
		}
	} else {
		// Load from environment variables
		cfg = loadConfigFromEnv(l)
	}

	// Create and start server
	server, err := webui.NewServer(cfg, l)
	if err != nil {
		l.Fatal("Failed to create server", zap.Error(err))
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		l.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		if err := server.Stop(); err != nil {
			l.Error("Failed to stop server", zap.Error(err))
		}
	}()

	// Start server
	if err := server.Start(); err != nil {
		l.Fatal("Failed to start server", zap.Error(err))
	}
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv(l *zap.Logger) *webui.Config {
	cfg := &webui.Config{}

	// Server configuration
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)

	// Core API configuration
	cfg.Core.URL = getEnv("CORE_URL", "http://ops-butler-core:8080")

	// Database configuration
	cfg.Database = &database.Config{
		Type:     getEnv("DB_TYPE", "sqlite"),
		Host:     getEnv("DB_HOST", ""),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", ""),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "ops_butler"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		Path:     getEnv("DB_PATH", "/data/ops-butler.db"),
	}

	return cfg
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
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsBool gets an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	if valueStr == "true" || valueStr == "1" || valueStr == "yes" {
		return true
	}
	if valueStr == "false" || valueStr == "0" || valueStr == "no" {
		return false
	}
	return defaultValue
}