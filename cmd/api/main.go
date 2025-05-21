package main

import (
	"log"
	"os"

	"github.com/BogdanDolia/ops-butler/internal/api"
	"github.com/BogdanDolia/ops-butler/internal/config"
	"github.com/BogdanDolia/ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize logger
	l, err := logger.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Sync()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		l.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		l.Fatal("Failed to run database migrations", zap.Error(err))
		os.Exit(1)
	}

	// Create repository
	repo := database.NewGormRepository(db)

	// Create and start server
	server := api.NewServer(cfg, l, repo)
	if err := server.Run(); err != nil {
		l.Fatal("Server error", zap.Error(err))
		os.Exit(1)
	}
}
