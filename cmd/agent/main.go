package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BogdanDolia/ops-butler/internal/agent"
	"github.com/BogdanDolia/ops-butler/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := agent.NewConfig()

	// Initialize logger
	l, err := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Sync()

	// Create agent
	a := agent.NewAgent(cfg, l)

	// Start agent
	if err := a.Start(); err != nil {
		l.Fatal("Failed to start agent", zap.Error(err))
		os.Exit(1)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Stop agent
	if err := a.Stop(); err != nil {
		l.Error("Failed to stop agent", zap.Error(err))
		os.Exit(1)
	}
}
