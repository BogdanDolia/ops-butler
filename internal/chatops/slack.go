package chatops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// SlackClient represents a Slack client
type SlackClient struct {
	config SlackConfig
	logger *zap.Logger
	client *slack.Client
}

// SlackInteractionPayload represents the payload from Slack interactive components
type SlackInteractionPayload struct {
	Type        string            `json:"type"`
	User        SlackUser         `json:"user"`
	APIAppID    string            `json:"api_app_id"`
	Token       string            `json:"token"`
	TriggerID   string            `json:"trigger_id"`
	Team        SlackTeam         `json:"team"`
	Channel     SlackChannel      `json:"channel"`
	Message     SlackMessage      `json:"message"`
	ResponseURL string            `json:"response_url"`
	Actions     []SlackAction     `json:"actions"`
	State       map[string]string `json:"state"`
}

type SlackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackTeam struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackMessage struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	User      string `json:"user"`
	Timestamp string `json:"ts"`
}

type SlackAction struct {
	Type     string `json:"type"`
	ActionID string `json:"action_id"`
	BlockID  string `json:"block_id"`
	Value    string `json:"value"`
	Text     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"text"`
	ActionTS string `json:"action_ts"`
}

// NewSlackClient creates a new Slack client
func NewSlackClient(config SlackConfig, logger *zap.Logger) (*SlackClient, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("slack is not enabled")
	}

	// In demo mode, we don't need a real token
	if config.DemoMode {
		logger.Info("Slack client initialized in demo mode")
		return &SlackClient{
			config: config,
			logger: logger,
			client: nil, // No real client in demo mode
		}, nil
	}

	if config.Token == "" {
		return nil, fmt.Errorf("slack token is required")
	}

	client := slack.New(config.Token)

	return &SlackClient{
		config: config,
		logger: logger,
		client: client,
	}, nil
}

// SendMessage sends a message to a Slack channel
func (s *SlackClient) SendMessage(channel, text string) (string, error) {
	s.logger.Debug("Sending message to Slack", zap.String("channel", channel), zap.String("text", text))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	// Demo mode - just log the message
	if s.config.DemoMode {
		s.logger.Info("📤 [DEMO MODE] Slack message",
			zap.String("channel", channel),
			zap.String("text", text))
		return fmt.Sprintf("demo_%d", time.Now().Unix()), nil
	}

	// Real Slack API call
	channelID, timestamp, err := s.client.PostMessage(channel, slack.MsgOptionText(text, false))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	s.logger.Debug("Message sent successfully",
		zap.String("channel_id", channelID),
		zap.String("timestamp", timestamp))

	return timestamp, nil
}

// SendReminderMessage sends a reminder message with interactive buttons
func (s *SlackClient) SendReminderMessage(channel, text string, taskID uint) (string, error) {
	s.logger.Debug("Sending reminder message to Slack",
		zap.String("channel", channel),
		zap.String("text", text),
		zap.Uint("task_id", taskID))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	// Demo mode - just log the message
	if s.config.DemoMode {
		s.logger.Info("📤 [DEMO MODE] Slack reminder message with buttons",
			zap.String("channel", channel),
			zap.String("text", text),
			zap.Uint("task_id", taskID),
			zap.Strings("buttons", []string{"🚀 Run Now", "⏰ Snooze 2h", "❌ Cancel"}))
		return fmt.Sprintf("demo_reminder_%d", time.Now().Unix()), nil
	}

	// Create Block Kit message with interactive buttons
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			nil,
			nil,
		),
		slack.NewActionBlock(
			"task_actions",
			slack.NewButtonBlockElement(
				"run_now",
				fmt.Sprintf("task_%d", taskID),
				slack.NewTextBlockObject(slack.PlainTextType, "🚀 Run Now", false, false),
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				"snooze_2h",
				fmt.Sprintf("task_%d", taskID),
				slack.NewTextBlockObject(slack.PlainTextType, "⏰ Snooze 2h", false, false),
			),
			slack.NewButtonBlockElement(
				"cancel_task",
				fmt.Sprintf("task_%d", taskID),
				slack.NewTextBlockObject(slack.PlainTextType, "❌ Cancel", false, false),
			).WithStyle(slack.StyleDanger),
		),
	}

	channelID, timestamp, err := s.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return "", fmt.Errorf("failed to send reminder message: %w", err)
	}

	s.logger.Debug("Reminder message sent successfully",
		zap.String("channel_id", channelID),
		zap.String("timestamp", timestamp),
		zap.Uint("task_id", taskID))

	return timestamp, nil
}

// SendTaskExecutionMessage sends a message about task execution with logs
func (s *SlackClient) SendTaskExecutionMessage(channel, text string, taskID uint, logs string) (string, error) {
	s.logger.Debug("Sending task execution message to Slack",
		zap.String("channel", channel),
		zap.Uint("task_id", taskID))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	// Truncate logs if too long for Slack
	const maxLogsLength = 2000
	displayLogs := logs
	if len(logs) > maxLogsLength {
		displayLogs = logs[:maxLogsLength] + "\n... (truncated)"
	}

	// Create Block Kit message with task execution results
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "*First 40 lines of logs:*", false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("```\n%s\n```", displayLogs), false, false),
			nil,
			nil,
		),
		slack.NewActionBlock(
			"task_execution_actions",
			slack.NewButtonBlockElement(
				"view_full_logs",
				fmt.Sprintf("task_%d", taskID),
				slack.NewTextBlockObject(slack.PlainTextType, "📋 View Full Logs", false, false),
			).WithStyle(slack.StylePrimary),
		),
	}

	channelID, timestamp, err := s.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return "", fmt.Errorf("failed to send task execution message: %w", err)
	}

	s.logger.Debug("Task execution message sent successfully",
		zap.String("channel_id", channelID),
		zap.String("timestamp", timestamp),
		zap.Uint("task_id", taskID))

	return timestamp, nil
}

// ScheduleMessage schedules a message to be sent at a future time
func (s *SlackClient) ScheduleMessage(channel, text string, postAt time.Time) (string, string, error) {
	s.logger.Debug("Scheduling message in Slack",
		zap.String("channel", channel),
		zap.String("text", text),
		zap.Time("post_at", postAt))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	// Convert time.Time to Unix timestamp string
	postAtUnix := fmt.Sprintf("%d", postAt.Unix())

	scheduledMessageID, _, err := s.client.ScheduleMessage(channel, postAtUnix, slack.MsgOptionText(text, false))
	if err != nil {
		return "", "", fmt.Errorf("failed to schedule message: %w", err)
	}

	timestamp := fmt.Sprintf("%d.%d", postAt.Unix(), 0)
	return scheduledMessageID, timestamp, nil
}

// UploadFile uploads a file to a Slack channel
func (s *SlackClient) UploadFile(channel, filename, content string) (string, error) {
	s.logger.Debug("Uploading file to Slack",
		zap.String("channel", channel),
		zap.String("filename", filename))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	file, err := s.client.UploadFile(slack.FileUploadParameters{
		Filename: filename,
		Title:    filename,
		Content:  content,
		Channels: []string{channel},
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	s.logger.Debug("File uploaded successfully",
		zap.String("file_id", file.ID),
		zap.String("filename", filename))

	return file.ID, nil
}

// HandleInteractiveComponent handles an interactive component from Slack
func (s *SlackClient) HandleInteractiveComponent(payload []byte) (*SlackInteractionPayload, error) {
	s.logger.Debug("Handling interactive component from Slack")

	var interactionPayload SlackInteractionPayload
	if err := json.Unmarshal(payload, &interactionPayload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	s.logger.Debug("Received interactive component",
		zap.String("type", interactionPayload.Type),
		zap.String("user", interactionPayload.User.Name),
		zap.String("channel", interactionPayload.Channel.Name),
		zap.Int("actions_count", len(interactionPayload.Actions)))

	return &interactionPayload, nil
}

// VerifyRequest verifies a request from Slack using the signing secret
func (s *SlackClient) VerifyRequest(r *http.Request, body []byte) error {
	if s.config.SigningSecret == "" {
		s.logger.Warn("Slack signing secret not configured, skipping verification")
		return nil
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing required headers")
	}

	// Check if the timestamp is within 5 minutes
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("request timestamp too old")
	}

	// Create the signature base string
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	// Calculate the expected signature
	mac := hmac.New(sha256.New, []byte(s.config.SigningSecret))
	mac.Write([]byte(sigBaseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// UpdateMessage updates an existing message in Slack
func (s *SlackClient) UpdateMessage(channel, timestamp, text string) error {
	s.logger.Debug("Updating message in Slack",
		zap.String("channel", channel),
		zap.String("timestamp", timestamp),
		zap.String("text", text))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	_, _, _, err := s.client.UpdateMessage(channel, timestamp, slack.MsgOptionText(text, false))
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	s.logger.Debug("Message updated successfully")
	return nil
}

// SendEphemeralMessage sends an ephemeral message to a user
func (s *SlackClient) SendEphemeralMessage(channel, user, text string) (string, error) {
	s.logger.Debug("Sending ephemeral message to Slack",
		zap.String("channel", channel),
		zap.String("user", user),
		zap.String("text", text))

	if channel == "" {
		channel = s.config.DefaultChannel
	}

	// Ensure channel starts with #
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "C") {
		channel = "#" + channel
	}

	timestamp, err := s.client.PostEphemeral(channel, user, slack.MsgOptionText(text, false))
	if err != nil {
		return "", fmt.Errorf("failed to send ephemeral message: %w", err)
	}

	s.logger.Debug("Ephemeral message sent successfully", zap.String("timestamp", timestamp))
	return timestamp, nil
}
