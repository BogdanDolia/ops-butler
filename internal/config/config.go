package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	Logging   LoggingConfig
	Telemetry TelemetryConfig
	ChatOps   ChatOpsConfig
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Host             string
	Port             int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ShutdownTimeout  time.Duration
	TLSEnabled       bool
	TLSCertFile      string
	TLSKeyFile       string
	CORSAllowOrigins []string
}

// DatabaseConfig holds the database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// AuthConfig holds the authentication configuration
type AuthConfig struct {
	JWTSecret        string
	JWTExpiryMinutes int
	OIDCEnabled      bool
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCScopes       []string
}

// LoggingConfig holds the logging configuration
type LoggingConfig struct {
	Level      string
	Format     string
	OutputPath string
}

// TelemetryConfig holds the telemetry configuration
type TelemetryConfig struct {
	MetricsEnabled  bool
	TracingEnabled  bool
	TracingEndpoint string
	ServiceName     string
}

// ChatOpsConfig holds the ChatOps configuration
type ChatOpsConfig struct {
	SlackEnabled       bool
	SlackToken         string
	SlackSigningSecret string
	GoogleChatEnabled  bool
	GoogleChatToken    string
}

// NewConfig creates a new configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:             getEnv("SERVER_HOST", "0.0.0.0"),
			Port:             getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:      getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:     getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout:  getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 5*time.Second),
			TLSEnabled:       getEnvAsBool("SERVER_TLS_ENABLED", false),
			TLSCertFile:      getEnv("SERVER_TLS_CERT_FILE", ""),
			TLSKeyFile:       getEnv("SERVER_TLS_KEY_FILE", ""),
			CORSAllowOrigins: getEnvAsSlice("SERVER_CORS_ALLOW_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "k8s_ops_portal"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			JWTSecret:        getEnv("AUTH_JWT_SECRET", "your-secret-key"),
			JWTExpiryMinutes: getEnvAsInt("AUTH_JWT_EXPIRY_MINUTES", 60),
			OIDCEnabled:      getEnvAsBool("AUTH_OIDC_ENABLED", false),
			OIDCIssuerURL:    getEnv("AUTH_OIDC_ISSUER_URL", ""),
			OIDCClientID:     getEnv("AUTH_OIDC_CLIENT_ID", ""),
			OIDCClientSecret: getEnv("AUTH_OIDC_CLIENT_SECRET", ""),
			OIDCRedirectURL:  getEnv("AUTH_OIDC_REDIRECT_URL", ""),
			OIDCScopes:       getEnvAsSlice("AUTH_OIDC_SCOPES", []string{"openid", "profile", "email"}),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
		},
		Telemetry: TelemetryConfig{
			MetricsEnabled:  getEnvAsBool("TELEMETRY_METRICS_ENABLED", true),
			TracingEnabled:  getEnvAsBool("TELEMETRY_TRACING_ENABLED", false),
			TracingEndpoint: getEnv("TELEMETRY_TRACING_ENDPOINT", ""),
			ServiceName:     getEnv("TELEMETRY_SERVICE_NAME", "ops-butler"),
		},
		ChatOps: ChatOpsConfig{
			SlackEnabled:       getEnvAsBool("CHATOPS_SLACK_ENABLED", false),
			SlackToken:         getEnv("CHATOPS_SLACK_TOKEN", ""),
			SlackSigningSecret: getEnv("CHATOPS_SLACK_SIGNING_SECRET", ""),
			GoogleChatEnabled:  getEnvAsBool("CHATOPS_GOOGLE_CHAT_ENABLED", false),
			GoogleChatToken:    getEnv("CHATOPS_GOOGLE_CHAT_TOKEN", ""),
		},
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

// getEnvAsSlice gets an environment variable as a slice or returns a default value
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// Address returns the server address
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
