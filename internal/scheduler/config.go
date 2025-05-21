package scheduler

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the scheduler configuration
type Config struct {
	PollingInterval    time.Duration
	MaxConcurrentTasks int
	RedisURL           string
	RedisPassword      string
	RedisDB            int
	LogLevel           string
	LogFormat          string
}

// NewConfig creates a new scheduler configuration from environment variables
func NewConfig() *Config {
	return &Config{
		PollingInterval:    getEnvAsDuration("SCHEDULER_POLLING_INTERVAL", 30*time.Second),
		MaxConcurrentTasks: getEnvAsInt("SCHEDULER_MAX_CONCURRENT_TASKS", 10),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvAsInt("REDIS_DB", 0),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
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

// getEnvAsDuration gets an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Scheduler Config: PollingInterval=%s, MaxConcurrentTasks=%d, RedisURL=%s",
		c.PollingInterval, c.MaxConcurrentTasks, c.RedisURL)
}
