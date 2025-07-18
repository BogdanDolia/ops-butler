package api

import (
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
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Check if this is a URL verification request
	if c.Request.Header.Get("Content-Type") == "application/json" {
		var challenge struct {
			Challenge string `json:"challenge"`
		}
		if err := c.ShouldBindJSON(&challenge); err == nil && challenge.Challenge != "" {
			s.logger.Debug("Slack URL verification", zap.String("challenge", challenge.Challenge))
			c.JSON(http.StatusOK, gin.H{"challenge": challenge.Challenge})
			return
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

		body = []byte(payload)
	}

	// Handle the interaction
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
