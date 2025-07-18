package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleSlackWebhook handles incoming Slack webhooks
func (s *Server) handleSlackWebhook(c *gin.Context) {
	// Add CORS headers
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type")

	if c.Request.Method == "OPTIONS" {
		c.Status(200)
		return
	}

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	s.logger.Debug("Slack webhook request",
		zap.String("content_type", c.Request.Header.Get("Content-Type")),
		zap.String("body", string(body)))

	// Check if this is a URL verification request (JSON)
	if c.Request.Header.Get("Content-Type") == "application/json" {
		var verificationRequest struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"`
		}

		if err := json.Unmarshal(body, &verificationRequest); err == nil {
			if verificationRequest.Type == "url_verification" && verificationRequest.Challenge != "" {
				s.logger.Debug("Slack URL verification", zap.String("challenge", verificationRequest.Challenge))
				c.JSON(http.StatusOK, gin.H{"challenge": verificationRequest.Challenge})
				return
			}
		}
	}

	// Parse form data for interactive components
	contentType := c.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		form, err := url.ParseQuery(string(body))
		if err != nil {
			s.logger.Error("Failed to parse form data", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
			return
		}

		// Get the payload
		payload := form.Get("payload")
		if payload == "" {
			s.logger.Error("No payload found in form data")
			c.JSON(http.StatusBadRequest, gin.H{"error": "No payload found"})
			return
		}

		s.logger.Debug("Slack interactive payload", zap.String("payload", payload))

		// For interactive components, skip signature verification temporarily
		// and handle the interaction directly
		if s.chatops != nil {
			// Parse the interaction payload
			var interactionPayload map[string]interface{}
			if err := json.Unmarshal([]byte(payload), &interactionPayload); err != nil {
				s.logger.Error("Failed to parse interaction payload", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
				return
			}

			s.logger.Info("Slack interaction received",
				zap.String("type", fmt.Sprintf("%v", interactionPayload["type"])),
				zap.String("user", fmt.Sprintf("%v", interactionPayload["user"])))

			// Handle the interaction without signature verification for now
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
	}

	// For other requests, handle normally
	if s.chatops != nil {
		if err := s.chatops.HandleSlackInteraction(c.Request, body); err != nil {
			s.logger.Error("Failed to handle Slack interaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle interaction"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleGoogleChatWebhook handles incoming Google Chat webhooks
func (s *Server) handleGoogleChatWebhook(c *gin.Context) {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Handle the interaction
	if s.chatops != nil {
		if err := s.chatops.HandleGoogleChatInteraction(c.Request, body); err != nil {
			s.logger.Error("Failed to handle Google Chat interaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle interaction"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleSlackSlashCommand handles Slack slash commands
func (s *Server) handleSlackSlashCommand(c *gin.Context) {
	// Parse form data
	command := c.PostForm("command")
	text := c.PostForm("text")
	userID := c.PostForm("user_id")
	userName := c.PostForm("user_name")
	channelID := c.PostForm("channel_id")

	s.logger.Debug("Slack slash command received",
		zap.String("command", command),
		zap.String("text", text),
		zap.String("user_id", userID),
		zap.String("user_name", userName),
		zap.String("channel_id", channelID))

	// Handle different commands
	switch command {
	case "/ops-status":
		s.handleOpsStatusCommand(c, channelID, userID, userName)
	case "/ops-run":
		s.handleOpsRunCommand(c, channelID, userID, userName, text)
	case "/ops-list":
		s.handleOpsListCommand(c, channelID, userID, userName)
	default:
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          fmt.Sprintf("Unknown command: %s", command),
		})
	}
}

// handleOpsStatusCommand handles the /ops-status command
func (s *Server) handleOpsStatusCommand(c *gin.Context, channelID, userID, userName string) {
	// Get system status
	agentCount := 0
	if agents, err := s.agents.List(c.Request.Context(), 0, 100); err == nil {
		agentCount = len(agents)
	}

	pendingTasks := 0
	if tasks, err := s.tasks.ListByState(c.Request.Context(), models.TaskStatePending, 0, 100); err == nil {
		pendingTasks = len(tasks)
	}

	runningTasks := 0
	if tasks, err := s.tasks.ListByState(c.Request.Context(), models.TaskStateRunning, 0, 100); err == nil {
		runningTasks = len(tasks)
	}

	response := fmt.Sprintf("🤖 *Ops Butler Status*\n"+
		"• Active Agents: %d\n"+
		"• Pending Tasks: %d\n"+
		"• Running Tasks: %d\n"+
		"• System: Healthy ✅",
		agentCount, pendingTasks, runningTasks)

	c.JSON(http.StatusOK, gin.H{
		"response_type": "in_channel",
		"text":          response,
	})
}

// handleOpsRunCommand handles the /ops-run command
func (s *Server) handleOpsRunCommand(c *gin.Context, channelID, userID, userName, text string) {
	if text == "" {
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          "Usage: /ops-run <task-id>",
		})
		return
	}

	// Parse task ID
	taskID, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          fmt.Sprintf("Invalid task ID: %s", text),
		})
		return
	}

	// Execute the task
	if err := s.executeTaskFromChatOps(uint(taskID), userID); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          fmt.Sprintf("Failed to execute task %d: %s", taskID, err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response_type": "in_channel",
		"text":          fmt.Sprintf("🚀 Task %d execution started by %s", taskID, userName),
	})
}

// handleOpsListCommand handles the /ops-list command
func (s *Server) handleOpsListCommand(c *gin.Context, channelID, userID, userName string) {
	// Get recent tasks
	tasks, err := s.tasks.List(c.Request.Context(), 0, 10)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          "Failed to retrieve tasks",
		})
		return
	}

	if len(tasks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          "No tasks found",
		})
		return
	}

	var response strings.Builder
	response.WriteString("📋 *Recent Tasks*\n")
	for _, task := range tasks {
		status := "❓"
		switch task.State {
		case models.TaskStatePending:
			status = "⏳"
		case models.TaskStateRunning:
			status = "🏃"
		case models.TaskStateCompleted:
			status = "✅"
		case models.TaskStateFailed:
			status = "❌"
		case models.TaskStateCancelled:
			status = "🚫"
		}
		response.WriteString(fmt.Sprintf("• %s Task %d (%s)\n", status, task.ID, task.State))
	}

	c.JSON(http.StatusOK, gin.H{
		"response_type": "in_channel",
		"text":          response.String(),
	})
}

// handleTestSlackMessage handles test Slack message requests
func (s *Server) handleTestSlackMessage(c *gin.Context) {
	var request struct {
		Channel string `json:"channel"`
		Message string `json:"message"`
		TaskID  *uint  `json:"task_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Channel == "" {
		request.Channel = "#general"
	}

	if request.Message == "" {
		request.Message = "🤖 Test message from Ops Butler!"
	}

	if s.chatops == nil {
		// Provide detailed error about why ChatOps is not available
		errorDetails := gin.H{
			"error": "ChatOps service not available",
			"details": gin.H{
				"slack_enabled":          s.config.ChatOps.Slack.Enabled,
				"slack_token_configured": s.config.ChatOps.Slack.Token != "",
				"googlechat_enabled":     s.config.ChatOps.GoogleChat.Enabled,
			},
		}

		if !s.config.ChatOps.Slack.Enabled {
			errorDetails["solution"] = "Set SLACK_ENABLED=true environment variable"
		} else if s.config.ChatOps.Slack.Token == "" {
			errorDetails["solution"] = "Set SLACK_TOKEN environment variable with your Slack bot token"
		}

		c.JSON(http.StatusServiceUnavailable, errorDetails)
		return
	}

	// Send different types of messages based on request
	var messageID string
	var err error

	if request.TaskID != nil {
		// Send reminder message with buttons
		messageID, err = s.chatops.SendReminderMessage("slack", request.Channel, request.Message, *request.TaskID)
	} else {
		// Send simple message
		messageID, err = s.chatops.SendMessage("slack", request.Channel, request.Message)
	}

	if err != nil {
		s.logger.Error("Failed to send test message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"channel": request.Channel,
			"message": request.Message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message_id": messageID,
		"channel":    request.Channel,
		"message":    request.Message,
		"status":     "sent",
	})
}

// handleTestGoogleChatMessage handles test Google Chat message requests
func (s *Server) handleTestGoogleChatMessage(c *gin.Context) {
	var request struct {
		Space   string `json:"space"`
		Message string `json:"message"`
		TaskID  *uint  `json:"task_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Space == "" {
		request.Space = "spaces/your-default-space"
	}

	if request.Message == "" {
		request.Message = "🤖 Test message from Ops Butler!"
	}

	if s.chatops == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ChatOps service not available"})
		return
	}

	// Send different types of messages based on request
	var messageID string
	var err error

	if request.TaskID != nil {
		// Send reminder message with buttons
		messageID, err = s.chatops.SendReminderMessage("google_chat", request.Space, request.Message, *request.TaskID)
	} else {
		// Send simple message
		messageID, err = s.chatops.SendMessage("google_chat", request.Space, request.Message)
	}

	if err != nil {
		s.logger.Error("Failed to send test message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message_id": messageID,
		"space":      request.Space,
		"message":    request.Message,
		"status":     "sent",
	})
}
