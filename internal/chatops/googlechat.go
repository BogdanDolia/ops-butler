package chatops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

// GoogleChatClient represents a Google Chat client
type GoogleChatClient struct {
	config  GoogleChatConfig
	logger  *zap.Logger
	service *chat.Service
}

// GoogleChatInteractionPayload represents the payload from Google Chat interactive components
type GoogleChatInteractionPayload struct {
	Type                      string            `json:"type"`
	EventTime                 string            `json:"eventTime"`
	Message                   GoogleChatMessage `json:"message"`
	User                      GoogleChatUser    `json:"user"`
	Space                     GoogleChatSpace   `json:"space"`
	Action                    GoogleChatAction  `json:"action"`
	Token                     string            `json:"token"`
	ConfigCompleteRedirectUrl string            `json:"configCompleteRedirectUrl"`
}

type GoogleChatMessage struct {
	Name         string           `json:"name"`
	Sender       GoogleChatUser   `json:"sender"`
	CreateTime   string           `json:"createTime"`
	Text         string           `json:"text"`
	Thread       GoogleChatThread `json:"thread"`
	Space        GoogleChatSpace  `json:"space"`
	ArgumentText string           `json:"argumentText"`
	Cards        []interface{}    `json:"cards"`
}

type GoogleChatUser struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Type        string `json:"type"`
}

type GoogleChatSpace struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

type GoogleChatThread struct {
	Name string `json:"name"`
}

type GoogleChatAction struct {
	ActionMethodName string                `json:"actionMethodName"`
	Parameters       []GoogleChatParameter `json:"parameters"`
}

type GoogleChatParameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewGoogleChatClient creates a new Google Chat client
func NewGoogleChatClient(config GoogleChatConfig, logger *zap.Logger) (*GoogleChatClient, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("google chat is not enabled")
	}

	if config.ServiceAccount == "" {
		return nil, fmt.Errorf("google chat service account is required")
	}

	// Create service account credentials
	creds, err := google.FindDefaultCredentials(context.Background(), chat.ChatBotScope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	// Create Chat service
	service, err := chat.NewService(context.Background(), option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create chat service: %w", err)
	}

	return &GoogleChatClient{
		config:  config,
		logger:  logger,
		service: service,
	}, nil
}

// SendMessage sends a message to a Google Chat space
func (g *GoogleChatClient) SendMessage(space, text string) (string, error) {
	g.logger.Debug("Sending message to Google Chat", zap.String("space", space), zap.String("text", text))

	if space == "" {
		space = g.config.DefaultSpace
	}

	message := &chat.Message{
		Text: text,
	}

	response, err := g.service.Spaces.Messages.Create(space, message).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	g.logger.Debug("Message sent successfully", zap.String("message_name", response.Name))
	return response.Name, nil
}

// SendReminderMessage sends a reminder message with interactive buttons
func (g *GoogleChatClient) SendReminderMessage(space, text string, taskID uint) (string, error) {
	g.logger.Debug("Sending reminder message to Google Chat",
		zap.String("space", space),
		zap.String("text", text),
		zap.Uint("task_id", taskID))

	if space == "" {
		space = g.config.DefaultSpace
	}

	// Create Cards v2 with interactive buttons
	card := &chat.GoogleAppsCardV1Card{
		Header: &chat.GoogleAppsCardV1CardHeader{
			Title:    "Task Reminder",
			Subtitle: fmt.Sprintf("Task ID: %d", taskID),
		},
		Sections: []*chat.GoogleAppsCardV1Section{
			{
				Widgets: []*chat.GoogleAppsCardV1Widget{
					{
						TextParagraph: &chat.GoogleAppsCardV1TextParagraph{
							Text: text,
						},
					},
					{
						ButtonList: &chat.GoogleAppsCardV1ButtonList{
							Buttons: []*chat.GoogleAppsCardV1Button{
								{
									Text: "🚀 Run Now",
									OnClick: &chat.GoogleAppsCardV1OnClick{
										Action: &chat.GoogleAppsCardV1Action{
											Function: "runTaskNow",
											Parameters: []*chat.GoogleAppsCardV1ActionParameter{
												{
													Key:   "taskId",
													Value: fmt.Sprintf("%d", taskID),
												},
											},
										},
									},
								},
								{
									Text: "⏰ Snooze 2h",
									OnClick: &chat.GoogleAppsCardV1OnClick{
										Action: &chat.GoogleAppsCardV1Action{
											Function: "snoozeTask",
											Parameters: []*chat.GoogleAppsCardV1ActionParameter{
												{
													Key:   "taskId",
													Value: fmt.Sprintf("%d", taskID),
												},
												{
													Key:   "duration",
													Value: "2h",
												},
											},
										},
									},
								},
								{
									Text: "❌ Cancel",
									OnClick: &chat.GoogleAppsCardV1OnClick{
										Action: &chat.GoogleAppsCardV1Action{
											Function: "cancelTask",
											Parameters: []*chat.GoogleAppsCardV1ActionParameter{
												{
													Key:   "taskId",
													Value: fmt.Sprintf("%d", taskID),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	message := &chat.Message{
		CardsV2: []*chat.CardWithId{
			{
				CardId: fmt.Sprintf("task_reminder_%d", taskID),
				Card:   card,
			},
		},
	}

	response, err := g.service.Spaces.Messages.Create(space, message).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send reminder message: %w", err)
	}

	g.logger.Debug("Reminder message sent successfully",
		zap.String("message_name", response.Name),
		zap.Uint("task_id", taskID))

	return response.Name, nil
}

// SendTaskExecutionMessage sends a message about task execution with logs
func (g *GoogleChatClient) SendTaskExecutionMessage(space, text string, taskID uint, logs string) (string, error) {
	g.logger.Debug("Sending task execution message to Google Chat",
		zap.String("space", space),
		zap.Uint("task_id", taskID))

	if space == "" {
		space = g.config.DefaultSpace
	}

	// Truncate logs if too long
	const maxLogsLength = 1000
	displayLogs := logs
	if len(logs) > maxLogsLength {
		displayLogs = logs[:maxLogsLength] + "\n... (truncated)"
	}

	// Create Cards v2 with task execution results
	card := &chat.GoogleAppsCardV1Card{
		Header: &chat.GoogleAppsCardV1CardHeader{
			Title:    "Task Execution Result",
			Subtitle: fmt.Sprintf("Task ID: %d", taskID),
		},
		Sections: []*chat.GoogleAppsCardV1Section{
			{
				Widgets: []*chat.GoogleAppsCardV1Widget{
					{
						TextParagraph: &chat.GoogleAppsCardV1TextParagraph{
							Text: text,
						},
					},
					{
						TextParagraph: &chat.GoogleAppsCardV1TextParagraph{
							Text: fmt.Sprintf("<b>First 40 lines of logs:</b><br><pre>%s</pre>", displayLogs),
						},
					},
					{
						ButtonList: &chat.GoogleAppsCardV1ButtonList{
							Buttons: []*chat.GoogleAppsCardV1Button{
								{
									Text: "📋 View Full Logs",
									OnClick: &chat.GoogleAppsCardV1OnClick{
										Action: &chat.GoogleAppsCardV1Action{
											Function: "viewFullLogs",
											Parameters: []*chat.GoogleAppsCardV1ActionParameter{
												{
													Key:   "taskId",
													Value: fmt.Sprintf("%d", taskID),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	message := &chat.Message{
		CardsV2: []*chat.CardWithId{
			{
				CardId: fmt.Sprintf("task_execution_%d", taskID),
				Card:   card,
			},
		},
	}

	response, err := g.service.Spaces.Messages.Create(space, message).Do()
	if err != nil {
		return "", fmt.Errorf("failed to send task execution message: %w", err)
	}

	g.logger.Debug("Task execution message sent successfully",
		zap.String("message_name", response.Name),
		zap.Uint("task_id", taskID))

	return response.Name, nil
}

// UploadFile uploads a file to a Google Chat space
func (g *GoogleChatClient) UploadFile(space, filename, content string) (string, error) {
	g.logger.Debug("Uploading file to Google Chat",
		zap.String("space", space),
		zap.String("filename", filename))

	if space == "" {
		space = g.config.DefaultSpace
	}

	// Create a text message with the file content
	message := &chat.Message{
		Text: fmt.Sprintf("📄 **%s**\n```\n%s\n```", filename, content),
	}

	response, err := g.service.Spaces.Messages.Create(space, message).Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	g.logger.Debug("File uploaded successfully",
		zap.String("message_name", response.Name),
		zap.String("filename", filename))

	return response.Name, nil
}

// HandleInteractiveComponent handles an interactive component from Google Chat
func (g *GoogleChatClient) HandleInteractiveComponent(payload []byte) (*GoogleChatInteractionPayload, error) {
	g.logger.Debug("Handling interactive component from Google Chat")

	var interactionPayload GoogleChatInteractionPayload
	if err := json.Unmarshal(payload, &interactionPayload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	g.logger.Debug("Received interactive component",
		zap.String("type", interactionPayload.Type),
		zap.String("user", interactionPayload.User.DisplayName),
		zap.String("space", interactionPayload.Space.DisplayName),
		zap.String("action", interactionPayload.Action.ActionMethodName))

	return &interactionPayload, nil
}

// VerifyRequest verifies a request from Google Chat
func (g *GoogleChatClient) VerifyRequest(r *http.Request) error {
	// Google Chat uses Bearer token authentication
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing authorization header")
	}

	// In a real implementation, you would verify the JWT token here
	// For now, we'll just check if the header is present
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("invalid authorization header format")
	}

	return nil
}

// UpdateMessage updates an existing message in Google Chat
func (g *GoogleChatClient) UpdateMessage(space, messageName, text string) error {
	g.logger.Debug("Updating message in Google Chat",
		zap.String("space", space),
		zap.String("message_name", messageName),
		zap.String("text", text))

	message := &chat.Message{
		Text: text,
	}

	_, err := g.service.Spaces.Messages.Update(messageName, message).Do()
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	g.logger.Debug("Message updated successfully")
	return nil
}
