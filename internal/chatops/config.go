package chatops

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the ChatOps configuration
type Config struct {
	Slack      SlackConfig
	GoogleChat GoogleChatConfig
}

// SlackConfig holds the Slack configuration
type SlackConfig struct {
	Enabled        bool
	Token          string
	SigningSecret  string
	AppID          string
	VerifyToken    string
	BotUserID      string
	DefaultChannel string
}

// GoogleChatConfig holds the Google Chat configuration
type GoogleChatConfig struct {
	Enabled        bool
	ServiceAccount string
	ProjectID      string
	DefaultSpace   string
}

// NewConfig creates a new ChatOps configuration from environment variables
func NewConfig() *Config {
	return &Config{
		Slack: SlackConfig{
			Enabled:        getEnvAsBool("SLACK_ENABLED", false),
			Token:          getEnv("SLACK_TOKEN", ""),
			SigningSecret:  getEnv("SLACK_SIGNING_SECRET", ""),
			AppID:          getEnv("SLACK_APP_ID", ""),
			VerifyToken:    getEnv("SLACK_VERIFY_TOKEN", ""),
			BotUserID:      getEnv("SLACK_BOT_USER_ID", ""),
			DefaultChannel: getEnv("SLACK_DEFAULT_CHANNEL", "general"),
		},
		GoogleChat: GoogleChatConfig{
			Enabled:        getEnvAsBool("GOOGLE_CHAT_ENABLED", false),
			ServiceAccount: getEnv("GOOGLE_CHAT_SERVICE_ACCOUNT", ""),
			ProjectID:      getEnv("GOOGLE_CHAT_PROJECT_ID", ""),
			DefaultSpace:   getEnv("GOOGLE_CHAT_DEFAULT_SPACE", ""),
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

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("ChatOps Config: Slack Enabled=%v, Google Chat Enabled=%v",
		c.Slack.Enabled, c.GoogleChat.Enabled)
}
