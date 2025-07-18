package chatops

import (
	"fmt"
	"os"
	"strconv"

	"github.com/BogdanDolia/ops-butler/internal/config"
)

// Re-export config types for convenience
type Config = config.ChatOpsConfig
type SlackConfig = config.SlackConfig
type GoogleChatConfig = config.GoogleChatConfig

// Config holds the ChatOps configuration
type Config struct {
	Slack      SlackConfig
	GoogleChat GoogleChatConfig
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

// FromConfigChatOps converts a config.ChatOpsConfig to chatops.Config
func FromConfigChatOps(cfg config.ChatOpsConfig) *Config {
	return &Config{
		Slack: SlackConfig{
			Enabled:        cfg.SlackEnabled,
			Token:          cfg.SlackToken,
			SigningSecret:  cfg.SlackSigningSecret,
			AppID:          "", // Not provided in config.ChatOpsConfig
			VerifyToken:    "", // Not provided in config.ChatOpsConfig
			BotUserID:      "", // Not provided in config.ChatOpsConfig
			DefaultChannel: "general",
		},
		GoogleChat: GoogleChatConfig{
			Enabled:        cfg.GoogleChatEnabled,
			ServiceAccount: "", // Not provided in config.ChatOpsConfig
			ProjectID:      "", // Not provided in config.ChatOpsConfig
			DefaultSpace:   "",
		},
	}
}
