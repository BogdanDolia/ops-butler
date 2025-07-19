package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds the database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewConfig creates a new database configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "k8s_ops_portal"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// DSN returns the database connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// Connect establishes a connection to the database
func Connect(config *Config) (*gorm.DB, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.Template{},
		&models.TaskInstance{},
		&models.Reminder{},
		&models.ExecutionLog{},
		&models.ClusterAgent{},
		&models.User{},
	)
	if err != nil {
		return err
	}

	// Seed default templates if they don't exist
	return SeedDefaultTemplates(db)
}

// SeedDefaultTemplates creates default templates if they don't exist
func SeedDefaultTemplates(db *gorm.DB) error {
	// Check if any templates exist
	var count int64
	db.Model(&models.Template{}).Count(&count)

	if count > 0 {
		// Templates already exist, no need to seed
		return nil
	}

	// Create default templates
	defaultTemplates := []models.Template{
		{
			Name:        "Check Pod Logs",
			Description: "Retrieve and check logs from a Kubernetes pod",
			Script: `#!/bin/bash
# Check pod logs
if [ -z "$POD_NAME" ]; then
    echo "Error: POD_NAME not provided"
    exit 1
fi

NAMESPACE=${NAMESPACE:-default}

echo "Checking logs for pod: $POD_NAME in namespace: $NAMESPACE"
kubectl logs $POD_NAME -n $NAMESPACE --tail=100
`,
			ParamsSchema: models.JSONSchema{
				"type": "object",
				"properties": map[string]interface{}{
					"podName": map[string]interface{}{
						"type":        "string",
						"description": "Name of the pod to check logs for",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace",
						"default":     "default",
					},
				},
				"required": []string{"podName"},
			},
			RequireApproval: false,
			CreatedBy:       1, // Default admin user
		},
		{
			Name:        "Restart Pod",
			Description: "Restart a Kubernetes pod by deleting it (if managed by a deployment)",
			Script: `#!/bin/bash
# Restart pod
if [ -z "$POD_NAME" ]; then
    echo "Error: POD_NAME not provided"
    exit 1
fi

NAMESPACE=${NAMESPACE:-default}

echo "Restarting pod: $POD_NAME in namespace: $NAMESPACE"
kubectl delete pod $POD_NAME -n $NAMESPACE
`,
			ParamsSchema: models.JSONSchema{
				"type": "object",
				"properties": map[string]interface{}{
					"podName": map[string]interface{}{
						"type":        "string",
						"description": "Name of the pod to restart",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace",
						"default":     "default",
					},
				},
				"required": []string{"podName"},
			},
			RequireApproval: true, // Require approval for destructive operations
			CreatedBy:       1,    // Default admin user
		},
	}

	for _, template := range defaultTemplates {
		if err := db.Create(&template).Error; err != nil {
			return fmt.Errorf("failed to create default template '%s': %w", template.Name, err)
		}
	}

	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
