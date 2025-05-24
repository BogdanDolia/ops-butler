package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/BogdanDolia/ops-butler/internal/chatops"
	"github.com/BogdanDolia/ops-butler/internal/config"
	"github.com/BogdanDolia/ops-butler/internal/database"
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
	chatops    *chatops.Service
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

	// TODO: Initialize chatops service
	// In a real implementation, we would initialize the chatops service here
	// server.chatops = chatops.NewService(cfg.ChatOps, log)

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
		}

		// WebSocket for real-time logs
		v1.GET("/ws/logs/:taskId", s.handleWebSocketLogs)
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
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Placeholder handlers for routes
func (s *Server) handleListTemplates(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "List templates"})
}

func (s *Server) handleGetTemplate(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Get template"})
}

func (s *Server) handleCreateTemplate(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Create template"})
}

func (s *Server) handleUpdateTemplate(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Update template"})
}

func (s *Server) handleDeleteTemplate(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Delete template"})
}

func (s *Server) handleListTasks(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "List tasks"})
}

func (s *Server) handleGetTask(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Get task"})
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
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Update task"})
}

func (s *Server) handleDeleteTask(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Delete task"})
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

		// TODO: Implement logic to get logs from the pod
		// For now, we'll just simulate getting logs
		logs := fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
		for i := 1; i <= 40; i++ {
			logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
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

		// TODO: Implement logic to get logs from the pod
		// For now, we'll just simulate getting logs
		logs := fmt.Sprintf("Simulated logs for pod %s in namespace %s\n", podName, namespace)
		for i := 1; i <= 40; i++ {
			logs += fmt.Sprintf("Log line %d: This is a simulated log line\n", i)
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
	// Since we don't have a proper agent repository implementation yet,
	// we'll return a mock list of agents
	agents := []gin.H{
		{
			"id":             1,
			"name":           "agent-1",
			"labels":         gin.H{"environment": "production", "region": "us-west-1"},
			"last_heartbeat": time.Now(),
			"status":         "active",
			"version":        "1.0.0",
		},
		{
			"id":             2,
			"name":           "agent-2",
			"labels":         gin.H{"environment": "staging", "region": "us-east-1"},
			"last_heartbeat": time.Now(),
			"status":         "active",
			"version":        "1.0.0",
		},
	}

	c.JSON(http.StatusOK, agents)
}

func (s *Server) handleGetAgent(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Get agent"})
}

func (s *Server) handleWebSocketLogs(c *gin.Context) {
	// TODO: Implement WebSocket handler
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket logs"})
}
