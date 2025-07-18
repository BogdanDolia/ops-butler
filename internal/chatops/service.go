package chatops

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// TaskActionHandler defines the interface for handling task actions
type TaskActionHandler interface {
	ExecuteTask(taskID uint, userID string) error
	SnoozeTask(taskID uint, duration string, userID string) error
	CancelTask(taskID uint, userID string) error
	GetTaskLogs(taskID uint) (string, error)
}

// Service represents a ChatOps service
type Service struct {
	config      *Config
	logger      *zap.Logger
	slackClient *SlackClient
	chatClient  *GoogleChatClient
	taskHandler TaskActionHandler
}

// NewService creates a new ChatOps service
func NewService(config *Config, logger *zap.Logger, taskHandler TaskActionHandler) (*Service, error) {
	service := &Service{
		config:      config,
		logger:      logger,
		taskHandler: taskHandler,
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

// SendTaskExecutionMessage sends a message about task execution with logs
func (s *Service) SendTaskExecutionMessage(platform, channel, text string, taskID uint, logs string) (string, error) {
	s.logger.Debug("Sending task execution message",
		zap.String("platform", platform),
		zap.String("channel", channel),
		zap.Uint("task_id", taskID))

	switch platform {
	case "slack":
		if s.slackClient == nil {
			return "", fmt.Errorf("slack is not enabled")
		}
		return s.slackClient.SendTaskExecutionMessage(channel, text, taskID, logs)
	case "google_chat":
		if s.chatClient == nil {
			return "", fmt.Errorf("google chat is not enabled")
		}
		return s.chatClient.SendTaskExecutionMessage(channel, text, taskID, logs)
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
	payload, err := s.slackClient.HandleInteractiveComponent(body)
	if err != nil {
		return fmt.Errorf("failed to handle interactive component: %w", err)
	}

	return s.processSlackAction(payload)
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
	payload, err := s.chatClient.HandleInteractiveComponent(body)
	if err != nil {
		return fmt.Errorf("failed to handle interactive component: %w", err)
	}

	return s.processGoogleChatAction(payload)
}

// processSlackAction processes a Slack action
func (s *Service) processSlackAction(payload *SlackInteractionPayload) error {
	if len(payload.Actions) == 0 {
		return fmt.Errorf("no actions in payload")
	}

	action := payload.Actions[0]
	userID := payload.User.ID

	// Extract task ID from action value
	taskIDStr := strings.TrimPrefix(action.Value, "task_")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	s.logger.Info("Processing Slack action",
		zap.String("action", action.ActionID),
		zap.Uint64("task_id", taskID),
		zap.String("user", userID))

	switch action.ActionID {
	case "run_now":
		return s.handleRunNow(uint(taskID), userID, "slack", payload.Channel.ID)
	case "snooze_2h":
		return s.handleSnooze(uint(taskID), "2h", userID, "slack", payload.Channel.ID)
	case "cancel_task":
		return s.handleCancel(uint(taskID), userID, "slack", payload.Channel.ID)
	case "view_full_logs":
		return s.handleViewLogs(uint(taskID), userID, "slack", payload.Channel.ID)
	default:
		return fmt.Errorf("unknown action: %s", action.ActionID)
	}
}

// processGoogleChatAction processes a Google Chat action
func (s *Service) processGoogleChatAction(payload *GoogleChatInteractionPayload) error {
	if len(payload.Action.Parameters) == 0 {
		return fmt.Errorf("no parameters in action")
	}

	userID := payload.User.Name
	actionName := payload.Action.ActionMethodName

	// Extract task ID from parameters
	var taskID uint
	for _, param := range payload.Action.Parameters {
		if param.Key == "taskId" {
			taskIDInt, err := strconv.ParseUint(param.Value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			taskID = uint(taskIDInt)
			break
		}
	}

	if taskID == 0 {
		return fmt.Errorf("task ID not found in parameters")
	}

	s.logger.Info("Processing Google Chat action",
		zap.String("action", actionName),
		zap.Uint("task_id", taskID),
		zap.String("user", userID))

	switch actionName {
	case "runTaskNow":
		return s.handleRunNow(taskID, userID, "google_chat", payload.Space.Name)
	case "snoozeTask":
		duration := "2h"
		for _, param := range payload.Action.Parameters {
			if param.Key == "duration" {
				duration = param.Value
				break
			}
		}
		return s.handleSnooze(taskID, duration, userID, "google_chat", payload.Space.Name)
	case "cancelTask":
		return s.handleCancel(taskID, userID, "google_chat", payload.Space.Name)
	case "viewFullLogs":
		return s.handleViewLogs(taskID, userID, "google_chat", payload.Space.Name)
	default:
		return fmt.Errorf("unknown action: %s", actionName)
	}
}

// handleRunNow handles the "run now" action
func (s *Service) handleRunNow(taskID uint, userID, platform, channel string) error {
	s.logger.Info("Executing task now",
		zap.Uint("task_id", taskID),
		zap.String("user", userID),
		zap.String("platform", platform))

	// Execute the task
	if err := s.taskHandler.ExecuteTask(taskID, userID); err != nil {
		s.logger.Error("Failed to execute task", zap.Error(err))
		_, err := s.SendMessage(platform, channel, fmt.Sprintf("❌ Failed to execute task %d: %s", taskID, err.Error()))
		return err
	}

	// Get task logs
	logs, err := s.taskHandler.GetTaskLogs(taskID)
	if err != nil {
		s.logger.Error("Failed to get task logs", zap.Error(err))
		logs = "Failed to retrieve logs"
	}

	// Send execution result
	text := fmt.Sprintf("✅ Task %d executed successfully by %s", taskID, userID)
	_, err = s.SendTaskExecutionMessage(platform, channel, text, taskID, logs)
	return err
}

// handleSnooze handles the "snooze" action
func (s *Service) handleSnooze(taskID uint, duration, userID, platform, channel string) error {
	s.logger.Info("Snoozing task",
		zap.Uint("task_id", taskID),
		zap.String("duration", duration),
		zap.String("user", userID),
		zap.String("platform", platform))

	if err := s.taskHandler.SnoozeTask(taskID, duration, userID); err != nil {
		s.logger.Error("Failed to snooze task", zap.Error(err))
		_, err := s.SendMessage(platform, channel, fmt.Sprintf("❌ Failed to snooze task %d: %s", taskID, err.Error()))
		return err
	}

	text := fmt.Sprintf("⏰ Task %d snoozed for %s by %s", taskID, duration, userID)
	_, err := s.SendMessage(platform, channel, text)
	return err
}

// handleCancel handles the "cancel" action
func (s *Service) handleCancel(taskID uint, userID, platform, channel string) error {
	s.logger.Info("Canceling task",
		zap.Uint("task_id", taskID),
		zap.String("user", userID),
		zap.String("platform", platform))

	if err := s.taskHandler.CancelTask(taskID, userID); err != nil {
		s.logger.Error("Failed to cancel task", zap.Error(err))
		_, err := s.SendMessage(platform, channel, fmt.Sprintf("❌ Failed to cancel task %d: %s", taskID, err.Error()))
		return err
	}

	text := fmt.Sprintf("❌ Task %d cancelled by %s", taskID, userID)
	_, err := s.SendMessage(platform, channel, text)
	return err
}

// handleViewLogs handles the "view logs" action
func (s *Service) handleViewLogs(taskID uint, userID, platform, channel string) error {
	s.logger.Info("Viewing task logs",
		zap.Uint("task_id", taskID),
		zap.String("user", userID),
		zap.String("platform", platform))

	logs, err := s.taskHandler.GetTaskLogs(taskID)
	if err != nil {
		s.logger.Error("Failed to get task logs", zap.Error(err))
		_, err := s.SendMessage(platform, channel, fmt.Sprintf("❌ Failed to retrieve logs for task %d: %s", taskID, err.Error()))
		return err
	}

	filename := fmt.Sprintf("task_%d_logs.txt", taskID)
	_, err = s.UploadFile(platform, channel, filename, logs)
	if err != nil {
		s.logger.Error("Failed to upload logs file", zap.Error(err))
		// Fallback to sending logs as text
		text := fmt.Sprintf("📋 Full logs for task %d:\n```\n%s\n```", taskID, logs)
		_, err = s.SendMessage(platform, channel, text)
	}

	return err
}
