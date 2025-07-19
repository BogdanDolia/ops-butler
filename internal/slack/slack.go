package slack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/models"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// Config represents Slack configuration
type Config struct {
	BotToken   string `json:"bot_token"`
	AppToken   string `json:"app_token"`
	SigningKey string `json:"signing_key"`
}

// Client represents a Slack client
type Client struct {
	api    *slack.Client
	logger *zap.Logger
	config *Config
}

// NewClient creates a new Slack client
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("slack configuration is required")
	}
	if cfg.BotToken == "" {
		return nil, errors.New("slack bot token is required")
	}

	api := slack.New(cfg.BotToken)
	return &Client{
		api:    api,
		logger: logger,
		config: cfg,
	}, nil
}

// SendMessage sends a message to a Slack channel
func (c *Client) SendMessage(channelID, text string, options ...slack.MsgOption) (string, string, error) {
	opts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}
	opts = append(opts, options...)

	channelID, timestamp, err := c.api.PostMessage(channelID, opts...)
	if err != nil {
		c.logger.Error("Failed to send message to Slack",
			zap.String("channel", channelID),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("failed to send message to Slack: %w", err)
	}

	return channelID, timestamp, nil
}

// SendTaskUpdate sends a task update to a Slack channel
func (c *Client) SendTaskUpdate(task *models.TaskInstance, text string) error {
	if task.SlackChannel == "" {
		return errors.New("task has no Slack channel")
	}

	options := []slack.MsgOption{}
	
	// If we have a thread, reply to it
	if task.SlackThread != "" {
		options = append(options, slack.MsgOptionTS(task.SlackThread))
	}

	// Add task status as an attachment
	attachment := slack.Attachment{
		Color: getColorForTaskState(task.State),
		Fields: []slack.AttachmentField{
			{
				Title: "Task ID",
				Value: fmt.Sprintf("%d", task.ID),
				Short: true,
			},
			{
				Title: "Status",
				Value: string(task.State),
				Short: true,
			},
		},
	}
	
	// Add task type and template if available
	if task.TaskType != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Type",
			Value: string(task.TaskType),
			Short: true,
		})
	}
	
	if task.Template != nil {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Template",
			Value: task.Template.Name,
			Short: true,
		})
	}

	options = append(options, slack.MsgOptionAttachments(attachment))

	_, timestamp, err := c.SendMessage(task.SlackChannel, text, options...)
	if err != nil {
		return err
	}

	// If this is the first message, save the thread timestamp
	if task.SlackThread == "" {
		task.SlackThread = timestamp
	}

	return nil
}

// ParseSlashCommand parses a Slack slash command
func (c *Client) ParseSlashCommand(command string, text string) (string, []string, error) {
	if !strings.HasPrefix(command, "/ops") {
		return "", nil, errors.New("invalid command prefix")
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "help", nil, nil
	}

	subCommand := parts[0]
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return subCommand, args, nil
}

// VerifySignature verifies the signature of a Slack request
func (c *Client) VerifySignature(signature, timestamp, body string) bool {
	return slack.ValidateSignature(signature, timestamp, body, c.config.SigningKey)
}

// getColorForTaskState returns a color for a task state
func getColorForTaskState(state models.TaskState) string {
	switch state {
	case models.TaskStatePending:
		return "#FFCC00" // Yellow
	case models.TaskStateScheduled:
		return "#3AA3E3" // Blue
	case models.TaskStateRunning:
		return "#3AA3E3" // Blue
	case models.TaskStateCompleted:
		return "#36A64F" // Green
	case models.TaskStateFailed:
		return "#FF0000" // Red
	case models.TaskStateCancelled:
		return "#808080" // Gray
	default:
		return "#FFFFFF" // White
	}
}