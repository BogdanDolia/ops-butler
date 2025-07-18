package agent

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the agent configuration
type Config struct {
	Name              string
	Labels            map[string]string
	ClusterName       string
	ServerAddress     string
	HeartbeatInterval time.Duration
	Namespace         string
	TLSEnabled        bool
	TLSCertFile       string
	TLSKeyFile        string
	TLSCAFile         string
	LogLevel          string
	LogFormat         string
}

// NewConfig creates a new agent configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Name:              getEnv("AGENT_NAME", getHostname()),
		Labels:            getEnvAsMap("CLUSTER_LABELS", map[string]string{}),
		ClusterName:       getEnv("CLUSTER_NAME", "default"),
		ServerAddress:     getEnv("API_SERVER", "localhost:8080"),
		HeartbeatInterval: getEnvAsDuration("HEARTBEAT_INTERVAL", 30*time.Second),
		Namespace:         getEnv("KUBERNETES_NAMESPACE", "default"),
		TLSEnabled:        getEnvAsBool("TLS_ENABLED", false),
		TLSCertFile:       getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:        getEnv("TLS_KEY_FILE", ""),
		TLSCAFile:         getEnv("TLS_CA_FILE", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		LogFormat:         getEnv("LOG_FORMAT", "json"),
	}
}

// getHostname gets the hostname of the machine
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsMap gets an environment variable as a map or returns a default value
func getEnvAsMap(key string, defaultValue map[string]string) map[string]string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	result := make(map[string]string)
	pairs := strings.Split(valueStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// getEnvAsBool gets an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
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
	return fmt.Sprintf("Agent Config: Name=%s, ClusterName=%s, Labels=%v, ServerAddress=%s, Namespace=%s",
		c.Name, c.ClusterName, c.Labels, c.ServerAddress, c.Namespace)
}
