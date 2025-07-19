package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/BogdanDolia/ops-butler/internal/chatops"
	"github.com/BogdanDolia/ops-butler/internal/config"
	"github.com/BogdanDolia/ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/internal/k8s"
	"github.com/BogdanDolia/ops-butler/internal/models"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	config     *config.Config
	logger     *zap.Logger
	db         *database.GormRepository
	templates  database.TemplateRepository
	tasks      database.TaskRepository
	agents     database.AgentRepository
	chatops    *chatops.Service
	k8sClient  *k8s.Client
	// Add other repositories as needed
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, log *zap.Logger, db *database.GormRepository) *Server {
	// Set Gin mode based on environment
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create server
	server := &Server{
		router: router,
		httpServer: &http.Server{
			Addr:         cfg.Server.Address(),
			Handler:      router,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
		config: cfg,
		logger: log,
		db:     db,
	}

	// Initialize repositories
	server.initRepositories(db)

	// Initialize Kubernetes client
	server.k8sClient = k8s.NewClient(log)
	log.Info("Kubernetes client initialized successfully")

	// Initialize chatops service
	log.Info("Initializing ChatOps service",
		zap.Bool("slack_enabled", cfg.ChatOps.Slack.Enabled),
		zap.Bool("googlechat_enabled", cfg.ChatOps.GoogleChat.Enabled),
		zap.String("slack_token_set", func() string {
			if cfg.ChatOps.Slack.Token != "" {
				return "yes"
			}
			return "no"
		}()),
		zap.Bool("slack_demo_mode", cfg.ChatOps.Slack.DemoMode))

	if cfg.ChatOps.Slack.Enabled || cfg.ChatOps.GoogleChat.Enabled {
		// Auto-enable demo mode if Slack is enabled but no token provided
		chatopsConfig := chatops.FromConfigChatOps(cfg.ChatOps)
		if cfg.ChatOps.Slack.Enabled && cfg.ChatOps.Slack.Token == "" && !cfg.ChatOps.Slack.DemoMode {
			log.Info("Auto-enabling Slack demo mode (no token provided)")
			chatopsConfig.Slack.DemoMode = true
		}

		chatopsService, err := chatops.NewService(chatopsConfig, log, server)
		if err != nil {
			log.Error("Failed to create ChatOps service", zap.Error(err))
		} else {
			server.chatops = chatopsService
			log.Info("ChatOps service initialized successfully")
		}
	} else {
		log.Warn("ChatOps service not enabled. Set SLACK_ENABLED=true or GOOGLE_CHAT_ENABLED=true to enable")
	}

	// Set up middleware
	server.setupMiddleware()

	// Set up routes
	server.setupRoutes()

	return server
}

// initRepositories initializes the repositories
func (s *Server) initRepositories(db *database.GormRepository) {
	// Initialize repositories
	s.templates = database.NewTemplateRepository(db.DB())
	s.tasks = database.NewTaskRepository(db.DB())
	s.agents = database.NewAgentRepository(db.DB())
	// Initialize other repositories as needed
}

// setupMiddleware sets up the middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(LoggerMiddleware(s.logger))

	// CORS middleware
	s.router.Use(CORSMiddleware(s.config.Server.CORSAllowOrigins))

	// Add other middleware as needed
}

// setupRoutes sets up the routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handleHealth)

	// Metrics endpoint
	if s.config.Telemetry.MetricsEnabled {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Templates
		templates := v1.Group("/templates")
		{
			templates.GET("", s.handleListTemplates)
			templates.GET("/:id", s.handleGetTemplate)
			templates.POST("", s.handleCreateTemplate)
			templates.PUT("/:id", s.handleUpdateTemplate)
			templates.DELETE("/:id", s.handleDeleteTemplate)
		}

		// Tasks
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", s.handleListTasks)
			tasks.GET("/:id", s.handleGetTask)
			tasks.POST("", s.handleCreateTask)
			tasks.PUT("/:id", s.handleUpdateTask)
			tasks.DELETE("/:id", s.handleDeleteTask)
			tasks.POST("/:id/execute", s.handleExecuteTask)
			tasks.GET("/:id/logs", s.handleGetTaskLogs)
		}

		// Agents
		agents := v1.Group("/agents")
		{
			agents.GET("", s.handleListAgents)
			agents.GET("/:id", s.handleGetAgent)
			agents.POST("/register", s.handleRegisterAgent)
			agents.POST("/heartbeat", s.handleAgentHeartbeat)
			agents.DELETE("/cleanup", s.handleCleanupInactiveAgents)
		}

		// WebSocket for real-time logs
		v1.GET("/ws/logs/:taskId", s.handleWebSocketLogs)

		// ChatOps webhooks
		chatops := v1.Group("/chatops")
		{
			chatops.POST("/slack/webhook", s.handleSlackWebhook)
			chatops.POST("/slack/slash", s.handleSlackSlashCommand)
			chatops.POST("/googlechat/webhook", s.handleGoogleChatWebhook)

			// Test endpoints
			chatops.POST("/test/slack", s.handleTestSlackMessage)
			chatops.POST("/test/googlechat", s.handleTestGoogleChatMessage)
		}
	}

	// Add other routes as needed
}

// Start starts the server
func (s *Server) Start() error {
	// Start the server in a goroutine
	go func() {
		s.logger.Info("Starting server", zap.String("address", s.config.Server.Address()))

		var err error
		if s.config.Server.TLSEnabled {
			err = s.httpServer.ListenAndServeTLS(s.config.Server.TLSCertFile, s.config.Server.TLSKeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	s.logger.Info("Stopping server")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown the server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("Server stopped")
	return nil
}

// Run runs the server until a signal is received
func (s *Server) Run() error {
	// Start the server
	if err := s.Start(); err != nil {
		return err
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Stop the server
	return s.Stop()
}

// LoggerMiddleware returns a gin middleware for logging requests
func LoggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		log.Info("Request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}

// CORSMiddleware returns a gin middleware for handling CORS
func CORSMiddleware(allowOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if the origin is allowed
		allowAll := contains(allowOrigins, "*")
		allowed := allowAll || contains(allowOrigins, origin)

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(c *gin.Context) {
	health := gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}

	// Check ChatOps status
	if s.chatops != nil {
		health["chatops"] = gin.H{
			"slack_enabled":      s.config.ChatOps.Slack.Enabled,
			"googlechat_enabled": s.config.ChatOps.GoogleChat.Enabled,
		}
	}

	c.JSON(http.StatusOK, health)
}

// Template management handlers
func (s *Server) handleListTemplates(c *gin.Context) {
	// Parse pagination parameters
	offset := 0
	limit := 10

	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	templates, err := s.templates.List(c.Request.Context(), offset, limit)
	if err != nil {
		s.logger.Error("Failed to list templates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
		"offset":    offset,
		"limit":     limit,
	})
}

func (s *Server) handleGetTemplate(c *gin.Context) {
	// Parse template ID from URL parameter
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	template, err := s.templates.GetByID(c.Request.Context(), uint(templateID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		s.logger.Error("Failed to get template", zap.Error(err), zap.Uint("id", uint(templateID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template"})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (s *Server) handleCreateTemplate(c *gin.Context) {
	var template models.Template
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if template.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template name is required"})
		return
	}

	if template.Script == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template script is required"})
		return
	}

	// Create the template
	err := s.templates.Create(c.Request.Context(), &template)
	if err != nil {
		s.logger.Error("Failed to create template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, template)
}

func (s *Server) handleUpdateTemplate(c *gin.Context) {
	// Parse template ID from URL parameter
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	// Check if template exists
	existingTemplate, err := s.templates.GetByID(c.Request.Context(), uint(templateID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		s.logger.Error("Failed to get template", zap.Error(err), zap.Uint("id", uint(templateID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template"})
		return
	}

	// Parse the updated template data
	var updateData models.Template
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the template fields
	existingTemplate.Name = updateData.Name
	existingTemplate.Description = updateData.Description
	existingTemplate.Script = updateData.Script
	existingTemplate.ParamsSchema = updateData.ParamsSchema
	existingTemplate.RequireApproval = updateData.RequireApproval

	// Validate required fields
	if existingTemplate.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template name is required"})
		return
	}

	if existingTemplate.Script == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template script is required"})
		return
	}

	err = s.templates.Update(c.Request.Context(), existingTemplate)
	if err != nil {
		s.logger.Error("Failed to update template", zap.Error(err), zap.Uint("id", uint(templateID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, existingTemplate)
}

func (s *Server) handleDeleteTemplate(c *gin.Context) {
	// Parse template ID from URL parameter
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	// Delete the template
	err = s.templates.Delete(c.Request.Context(), uint(templateID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		s.logger.Error("Failed to delete template", zap.Error(err), zap.Uint("id", uint(templateID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

func (s *Server) handleListTasks(c *gin.Context) {
	// Parse pagination parameters
	offset := 0
	limit := 10

	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	tasks, err := s.tasks.List(c.Request.Context(), offset, limit)
	if err != nil {
		s.logger.Error("Failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"count":  len(tasks),
		"offset": offset,
		"limit":  limit,
	})
}

func (s *Server) handleGetTask(c *gin.Context) {
	// Parse task ID from URL parameter
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := s.tasks.GetByID(c.Request.Context(), uint(taskID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		s.logger.Error("Failed to get task", zap.Error(err), zap.Uint("id", uint(taskID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) handleCreateTask(c *gin.Context) {
	var task models.TaskInstance
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values if not provided
	if task.State == "" {
		task.State = models.TaskStatePending
	}
	if task.Origin == "" {
		task.Origin = models.TaskOriginWeb
	}

	// Validate TemplateID if provided
	if task.TemplateID != nil && *task.TemplateID != 0 {
		// Check if the template exists
		_, err := s.templates.GetByID(c.Request.Context(), *task.TemplateID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Template with ID %d not found", *task.TemplateID)})
				return
			}
			s.logger.Error("Failed to validate template", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate template"})
			return
		}
	} else {
		// For task types that don't require a template, set TemplateID to nil
		task.TemplateID = nil
	}

	// Set agent_id to nil to avoid foreign key constraint violation
	// This is a temporary fix until we implement proper agent validation
	task.AgentID = nil

	// Create the task
	err := s.tasks.Create(c.Request.Context(), &task)
	if err != nil {
		s.logger.Error("Failed to create task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (s *Server) handleUpdateTask(c *gin.Context) {
	// Parse task ID from URL parameter
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Check if task exists
	existingTask, err := s.tasks.GetByID(c.Request.Context(), uint(taskID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		s.logger.Error("Failed to get task", zap.Error(err), zap.Uint("id", uint(taskID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	// Parse the updated task data
	var updateData models.TaskInstance
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update only the fields that are allowed to be modified
	if updateData.State != "" {
		existingTask.State = updateData.State
	}
	if updateData.DueAt != nil {
		existingTask.DueAt = updateData.DueAt
	}
	if updateData.Params != nil {
		existingTask.Params = updateData.Params
	}
	if updateData.AgentID != nil {
		existingTask.AgentID = updateData.AgentID
	}
	if updateData.ApprovedBy != nil {
		existingTask.ApprovedBy = updateData.ApprovedBy
	}
	if updateData.ApprovedAt != nil {
		existingTask.ApprovedAt = updateData.ApprovedAt
	}

	// Validate and update TemplateID if provided
	if updateData.TemplateID != nil {
		if *updateData.TemplateID == 0 {
			// Setting to nil/null
			existingTask.TemplateID = nil
		} else {
			// Validate that template exists
			_, err := s.templates.GetByID(c.Request.Context(), *updateData.TemplateID)
			if err != nil {
				if errors.Is(err, database.ErrNotFound) {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Template with ID %d not found", *updateData.TemplateID)})
					return
				}
				s.logger.Error("Failed to validate template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate template"})
				return
			}
			existingTask.TemplateID = updateData.TemplateID
		}
	}

	// Update completion timestamp if task is being marked as completed/failed/cancelled
	if updateData.State == models.TaskStateCompleted || updateData.State == models.TaskStateFailed || updateData.State == models.TaskStateCancelled {
		now := time.Now()
		existingTask.CompletedAt = &now
	}

	// Save the updated task
	err = s.tasks.Update(c.Request.Context(), existingTask)
	if err != nil {
		s.logger.Error("Failed to update task", zap.Error(err), zap.Uint("id", uint(taskID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, existingTask)
}

func (s *Server) handleDeleteTask(c *gin.Context) {
	// Parse task ID from URL parameter
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Delete the task
	err = s.tasks.Delete(c.Request.Context(), uint(taskID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		s.logger.Error("Failed to delete task", zap.Error(err), zap.Uint("id", uint(taskID)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func (s *Server) handleExecuteTask(c *gin.Context) {
	// Get task ID from URL
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Convert task ID to uint
	var id uint
	if _, err := fmt.Sscanf(taskID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get task from database
	task, err := s.tasks.GetByID(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	// Check if task is a "check logs" task
	if task.TaskType == models.TaskTypeCheckLogs {
		// Get pod name and namespace from task parameters
		podName, ok := task.Params["podName"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pod name is required"})
			return
		}

		namespace, ok := task.Params["namespace"].(string)
		if !ok {
			namespace = "default" // Default namespace
		}

		// Update task state
		task.State = models.TaskStateRunning
		if err := s.tasks.Update(c.Request.Context(), task); err != nil {
			s.logger.Error("Failed to update task state", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task state"})
			return
		}

		// Get logs from the pod using Kubernetes client
		logOpts := &k8s.LogOptions{
			Lines:    100, // Get last 100 lines
			Follow:   false,
			Previous: false,
		}

		var logs string

		// Check if pod exists first
		exists, err := s.k8sClient.CheckPodExists(c.Request.Context(), namespace, podName)
		if err != nil {
			s.logger.Error("Failed to check pod existence", zap.Error(err))
			logs = fmt.Sprintf("Error checking pod existence: %v\n", err)
			logs += fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
			for i := 1; i <= 40; i++ {
				logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
			}
		} else if !exists {
			s.logger.Warn("Pod does not exist", zap.String("pod", podName), zap.String("namespace", namespace))
			logs = fmt.Sprintf("Pod %s not found in namespace %s\n", podName, namespace)
			logs += "Simulated logs (pod not found):\n"
			for i := 1; i <= 40; i++ {
				logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
			}
		} else {
			logs, err = s.k8sClient.GetPodLogs(c.Request.Context(), namespace, podName, logOpts)
			if err != nil {
				s.logger.Error("Failed to get pod logs", zap.Error(err))
				// Fallback to simulated logs if real logs fail
				logs = fmt.Sprintf("Error getting real logs: %v\n", err)
				logs += fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
				for i := 1; i <= 40; i++ {
					logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
				}
			}
		}

		// Create a reminder for the task
		reminder := &models.Reminder{
			TaskID:   task.ID,
			ChatAt:   time.Now(),
			State:    models.ReminderStatePending,
			ChatType: task.Params["chatType"].(string),
			ChatID:   task.Params["chatId"].(string),
		}

		// TODO: Create a reminder repository and use it to create the reminder
		// For now, we'll just simulate creating a reminder
		s.logger.Info("Simulating creating a reminder", zap.Any("reminder", reminder))

		// TODO: Use the chatops service to send a notification with a button to check logs
		// In a real implementation, we would use the chatops service to send a notification
		// with a button to check logs. When the button is clicked, it would display the logs.
		// For example:
		// s.chatops.SendReminderMessage(reminder.ChatType, reminder.ChatID,
		//    fmt.Sprintf("Check logs for pod %s in namespace %s", podName, namespace), task.ID)

		// Update task state
		task.State = models.TaskStateCompleted
		task.CompletedAt = &time.Time{}
		*task.CompletedAt = time.Now()
		if err := s.tasks.Update(c.Request.Context(), task); err != nil {
			s.logger.Error("Failed to update task state", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task state"})
			return
		}

		// Return the logs
		c.JSON(http.StatusOK, gin.H{
			"message": "Task executed successfully",
			"logs":    logs,
		})
		return
	}

	// For other task types, just return a success message
	c.JSON(http.StatusOK, gin.H{"message": "Task executed successfully"})
}

func (s *Server) handleGetTaskLogs(c *gin.Context) {
	// Get task ID from URL
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Convert task ID to uint
	var id uint
	if _, err := fmt.Sscanf(taskID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get task from database
	task, err := s.tasks.GetByID(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	// Check if task is a "check logs" task
	if task.TaskType == models.TaskTypeCheckLogs {
		// Get pod name and namespace from task parameters
		podName, ok := task.Params["podName"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pod name is required"})
			return
		}

		namespace, ok := task.Params["namespace"].(string)
		if !ok {
			namespace = "default" // Default namespace
		}

		// Get logs from the pod using Kubernetes client
		logOpts := &k8s.LogOptions{
			Lines:    200, // Get last 200 lines for manual log retrieval
			Follow:   false,
			Previous: false,
		}

		var logs string

		// Check if pod exists first
		exists, err := s.k8sClient.CheckPodExists(c.Request.Context(), namespace, podName)
		if err != nil {
			s.logger.Error("Failed to check pod existence", zap.Error(err))
			logs = fmt.Sprintf("Error checking pod existence: %v\n", err)
			logs += fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
			for i := 1; i <= 40; i++ {
				logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
			}
		} else if !exists {
			s.logger.Warn("Pod does not exist", zap.String("pod", podName), zap.String("namespace", namespace))
			logs = fmt.Sprintf("Pod %s not found in namespace %s\n", podName, namespace)
			logs += "Simulated logs (pod not found):\n"
			for i := 1; i <= 40; i++ {
				logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
			}
		} else {
			logs, err = s.k8sClient.GetPodLogs(c.Request.Context(), namespace, podName, logOpts)
			if err != nil {
				s.logger.Error("Failed to get pod logs", zap.Error(err))
				// Fallback to simulated logs if real logs fail
				logs = fmt.Sprintf("Error getting real logs: %v\n", err)
				logs += fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
				for i := 1; i <= 40; i++ {
					logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
				}
			}
		}

		// Return the logs
		c.JSON(http.StatusOK, gin.H{
			"message": "Task logs retrieved successfully",
			"logs":    logs,
		})
		return
	}

	// For other task types, just return a success message
	c.JSON(http.StatusOK, gin.H{"message": "Task logs retrieved successfully"})
}

func (s *Server) handleListAgents(c *gin.Context) {
	// Parse query parameters
	showInactive := c.Query("show_inactive") == "true"

	// Get all agents from the repository
	agents, err := s.agents.List(c.Request.Context(), 0, 100) // Increased limit for now
	if err != nil {
		s.logger.Error("Failed to list agents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agents"})
		return
	}

	// Filter and update agent status based on heartbeat activity
	activeAgents := make([]*models.ClusterAgent, 0)
	now := time.Now()
	heartbeatThreshold := 5 * time.Minute // Consider agents inactive if no heartbeat in 5 minutes

	for _, agent := range agents {
		// Check if agent is active based on last heartbeat
		timeSinceLastHeartbeat := now.Sub(agent.LastHeartbeat)

		// Update agent status based on heartbeat recency
		wasActive := agent.Status == "active" || agent.Status == "healthy"
		isCurrentlyActive := timeSinceLastHeartbeat <= heartbeatThreshold

		if isCurrentlyActive && !wasActive {
			// Agent came back online, update status
			agent.Status = "active"
			s.agents.Update(c.Request.Context(), agent)
		} else if !isCurrentlyActive && wasActive {
			// Agent went offline, update status
			agent.Status = "inactive"
			s.agents.Update(c.Request.Context(), agent)
		} else if !isCurrentlyActive {
			// Ensure status reflects inactivity
			agent.Status = "inactive"
		}

		// Include agent in response based on filter
		if isCurrentlyActive || showInactive {
			activeAgents = append(activeAgents, agent)
		}
	}

	// Add metadata about filtering
	response := gin.H{
		"agents": activeAgents,
		"count":  len(activeAgents),
		"total":  len(agents),
		"filter": map[string]interface{}{
			"show_inactive":       showInactive,
			"heartbeat_threshold": heartbeatThreshold.String(),
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleGetAgent(c *gin.Context) {
	// Get agent ID from URL
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Convert agent ID to uint
	var id uint
	if _, err := fmt.Sscanf(agentID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get agent from repository
	agent, err := s.agents.GetByID(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

func (s *Server) handleWebSocketLogs(c *gin.Context) {
	// TODO: Implement WebSocket handler
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket logs"})
}

// handleRegisterAgent handles agent registration
func (s *Server) handleRegisterAgent(c *gin.Context) {
	var request struct {
		Name        string                 `json:"name" binding:"required"`
		ClusterName string                 `json:"cluster_name" binding:"required"`
		Labels      map[string]interface{} `json:"labels"`
		Version     string                 `json:"version"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		s.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check if agent already exists
	existingAgent, err := s.agents.GetByName(c.Request.Context(), request.Name+"-"+request.ClusterName)
	if err != nil && err != database.ErrNotFound {
		s.logger.Error("Failed to check for existing agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing agent"})
		return
	}

	// If agent exists, return its ID
	if existingAgent != nil {
		// Update the agent's status and last heartbeat
		existingAgent.Status = "active"
		existingAgent.LastHeartbeat = time.Now()
		existingAgent.Labels = models.JSONSchema(request.Labels)
		existingAgent.Version = request.Version

		if err := s.agents.Update(c.Request.Context(), existingAgent); err != nil {
			s.logger.Error("Failed to update existing agent", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing agent"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"agent_id": existingAgent.ID,
			"message":  "Agent already registered",
		})
		return
	}

	// Create a new agent
	agent := &models.ClusterAgent{
		Name:          request.Name + "-" + request.ClusterName,
		Labels:        models.JSONSchema(request.Labels),
		LastHeartbeat: time.Now(),
		Status:        "active",
		Version:       request.Version,
	}

	if err := s.agents.Create(c.Request.Context(), agent); err != nil {
		s.logger.Error("Failed to create agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	s.logger.Info("Agent registered",
		zap.String("name", agent.Name),
		zap.Any("labels", agent.Labels),
		zap.Uint("id", agent.ID))

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agent.ID,
		"message":  "Agent registered successfully",
	})
}

// handleAgentHeartbeat handles agent heartbeats
func (s *Server) handleAgentHeartbeat(c *gin.Context) {
	var request struct {
		AgentID string                 `json:"agent_id" binding:"required"`
		Labels  map[string]interface{} `json:"labels"`
		Status  string                 `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		s.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Parse agent ID
	var agentID uint
	if _, err := fmt.Sscanf(request.AgentID, "%d", &agentID); err != nil {
		// Try to get agent by name
		agent, err := s.agents.GetByName(c.Request.Context(), request.AgentID)
		if err != nil {
			s.logger.Error("Failed to get agent by name", zap.Error(err), zap.String("name", request.AgentID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID or name"})
			return
		}
		agentID = agent.ID
	}

	// Get agent from database
	agent, err := s.agents.GetByID(c.Request.Context(), agentID)
	if err != nil {
		s.logger.Error("Failed to get agent", zap.Error(err), zap.Uint("agent_id", agentID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Update agent status and last heartbeat
	agent.Status = request.Status
	agent.LastHeartbeat = time.Now()
	if request.Labels != nil {
		agent.Labels = models.JSONSchema(request.Labels)
	}

	if err := s.agents.Update(c.Request.Context(), agent); err != nil {
		s.logger.Error("Failed to update agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	s.logger.Debug("Heartbeat received",
		zap.Uint("agent_id", agent.ID),
		zap.String("name", agent.Name),
		zap.String("status", agent.Status))

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat received"})
}

// Implement TaskActionHandler interface
func (s *Server) ExecuteTask(taskID uint, userID string) error {
	// Get task from database
	task, err := s.tasks.GetByID(context.Background(), taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task state
	task.State = models.TaskStateRunning
	if err := s.tasks.Update(context.Background(), task); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	// Execute task logic (simulate for now)
	go func() {
		time.Sleep(2 * time.Second)
		task.State = models.TaskStateCompleted
		now := time.Now()
		task.CompletedAt = &now
		s.tasks.Update(context.Background(), task)
	}()

	return nil
}

func (s *Server) SnoozeTask(taskID uint, duration string, userID string) error {
	// Parse duration
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Get task from database
	task, err := s.tasks.GetByID(context.Background(), taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update due_at time
	newDueAt := time.Now().Add(dur)
	task.DueAt = &newDueAt
	if err := s.tasks.Update(context.Background(), task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (s *Server) CancelTask(taskID uint, userID string) error {
	// Get task from database
	task, err := s.tasks.GetByID(context.Background(), taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task state
	task.State = models.TaskStateCancelled
	if err := s.tasks.Update(context.Background(), task); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	return nil
}

func (s *Server) GetTaskLogs(taskID uint) (string, error) {
	// Get task from database
	task, err := s.tasks.GetByID(context.Background(), taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

	// Simulate getting logs
	logs := fmt.Sprintf("=== Task %d Execution Logs ===\n", taskID)
	logs += fmt.Sprintf("Started at: %s\n", task.CreatedAt.Format(time.RFC3339))
	logs += "Task Type: " + string(task.TaskType) + "\n"
	logs += "Parameters: " + fmt.Sprintf("%+v", task.Params) + "\n"
	logs += "State: " + string(task.State) + "\n"

	// Add simulated kubectl logs
	if task.TaskType == models.TaskTypeCheckLogs {
		if podName, ok := task.Params["podName"].(string); ok {
			namespace, _ := task.Params["namespace"].(string)
			if namespace == "" {
				namespace = "default"
			}

			logs += fmt.Sprintf("kubectl logs %s -n %s\n", podName, namespace)
			logs += fmt.Sprintf("--- Pod %s logs ---\n", podName)
			for i := 1; i <= 50; i++ {
				logs += fmt.Sprintf("[%s] Log line %d: Application running normally\n",
					time.Now().Add(-time.Duration(i)*time.Second).Format("2006-01-02 15:04:05"), i)
			}
		}
	}

	return logs, nil
}

// executeTaskFromChatOps executes a task from ChatOps
func (s *Server) executeTaskFromChatOps(taskID uint, userID string) error {
	return s.ExecuteTask(taskID, userID)
}

func (s *Server) handleCleanupInactiveAgents(c *gin.Context) {
	// Get cleanup threshold from query parameter (default: 1 hour)
	thresholdStr := c.DefaultQuery("threshold", "1h")
	threshold, err := time.ParseDuration(thresholdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid threshold format"})
		return
	}

	// Get all agents
	agents, err := s.agents.List(c.Request.Context(), 0, 1000)
	if err != nil {
		s.logger.Error("Failed to list agents for cleanup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agents"})
		return
	}

	// Find agents to cleanup
	var agentsToDelete []uint
	now := time.Now()

	for _, agent := range agents {
		timeSinceLastHeartbeat := now.Sub(agent.LastHeartbeat)
		if timeSinceLastHeartbeat > threshold {
			agentsToDelete = append(agentsToDelete, agent.ID)
		}
	}

	// Delete inactive agents
	deletedCount := 0
	for _, agentID := range agentsToDelete {
		if err := s.agents.Delete(c.Request.Context(), agentID); err != nil {
			s.logger.Error("Failed to delete inactive agent",
				zap.Error(err),
				zap.Uint("agent_id", agentID))
		} else {
			deletedCount++
			s.logger.Info("Deleted inactive agent",
				zap.Uint("agent_id", agentID))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Cleanup completed",
		"threshold":        thresholdStr,
		"agents_found":     len(agents),
		"agents_deleted":   deletedCount,
		"agents_remaining": len(agents) - deletedCount,
	})
}
