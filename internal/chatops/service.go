package chatops

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Service represents a ChatOps service
type Service struct {
	config      *Config
	logger      *zap.Logger
	slackClient *SlackClient
	chatClient  *GoogleChatClient
}

// NewService creates a new ChatOps service
func NewService(config *Config, logger *zap.Logger) (*Service, error) {
	service := &Service{
		config: config,
		logger: logger,
	}

	// Initialize Slack client if enabled
	if config.Slack.Enabled {
		slackClient, err := NewSlackClient(config.Slack, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Slack client: %w", err)
		}
		service.slackClient = slackClient
	}

	// Initialize Google Chat client if enabled
	if config.GoogleChat.Enabled {
		chatClient, err := NewGoogleChatClient(config.GoogleChat, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Chat client: %w", err)
		}
		service.chatClient = chatClient
	}

	return service, nil
}

// SendMessage sends a message to a channel or space
func (s *Service) SendMessage(platform, channel, text string) (string, error) {
	s.logger.Debug("Sending message",
		zap.String("platform", platform),
		zap.String("channel", channel),
		zap.String("text", text))

	switch platform {
	case "slack":
		if s.slackClient == nil {
			return "", fmt.Errorf("slack is not enabled")
		}
		return s.slackClient.SendMessage(channel, text)
	case "google_chat":
		if s.chatClient == nil {
			return "", fmt.Errorf("google chat is not enabled")
		}
		return s.chatClient.SendMessage(channel, text)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// SendReminderMessage sends a reminder message with interactive buttons
func (s *Service) SendReminderMessage(platform, channel, text string, taskID uint) (string, error) {
	s.logger.Debug("Sending reminder message",
		zap.String("platform", platform),
		zap.String("channel", channel),
		zap.String("text", text),
		zap.Uint("task_id", taskID))

	switch platform {
	case "slack":
		if s.slackClient == nil {
			return "", fmt.Errorf("slack is not enabled")
		}
		return s.slackClient.SendReminderMessage(channel, text, taskID)
	case "google_chat":
		if s.chatClient == nil {
			return "", fmt.Errorf("google chat is not enabled")
		}
		return s.chatClient.SendReminderMessage(channel, text, taskID)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// UploadFile uploads a file to a channel or space
func (s *Service) UploadFile(platform, channel, filename, content string) (string, error) {
	s.logger.Debug("Uploading file",
		zap.String("platform", platform),
		zap.String("channel", channel),
		zap.String("filename", filename))

	switch platform {
	case "slack":
		if s.slackClient == nil {
			return "", fmt.Errorf("slack is not enabled")
		}
		return s.slackClient.UploadFile(channel, filename, content)
	case "google_chat":
		if s.chatClient == nil {
			return "", fmt.Errorf("google chat is not enabled")
		}
		return s.chatClient.UploadFile(channel, filename, content)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// HandleSlackInteraction handles an interaction from Slack
func (s *Service) HandleSlackInteraction(r *http.Request, body []byte) error {
	if s.slackClient == nil {
		return fmt.Errorf("slack is not enabled")
	}

	// Verify the request
	if err := s.slackClient.VerifyRequest(r, body); err != nil {
		return fmt.Errorf("failed to verify request: %w", err)
	}

	// Handle the interaction
	return s.slackClient.HandleInteractiveComponent(body)
}

// HandleGoogleChatInteraction handles an interaction from Google Chat
func (s *Service) HandleGoogleChatInteraction(r *http.Request, body []byte) error {
	if s.chatClient == nil {
		return fmt.Errorf("google chat is not enabled")
	}

	// Verify the request
	if err := s.chatClient.VerifyRequest(r); err != nil {
		return fmt.Errorf("failed to verify request: %w", err)
	}

	// Handle the interaction
	return s.chatClient.HandleInteractiveComponent(body)
}
