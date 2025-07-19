package webui

import (
	"fmt"
	"net/http"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Config represents the configuration for the Web UI server
type Config struct {
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
	Core struct {
		URL string `json:"url"`
	} `json:"core"`
	Database *database.Config `json:"database"`
}

// Server represents the Web UI server
type Server struct {
	config *Config
	logger *zap.Logger
	db     database.Repository
	echo   *echo.Echo
}

// NewServer creates a new Web UI server
func NewServer(cfg *Config, logger *zap.Logger) (*Server, error) {
	// Create database repository
	db, err := database.NewRepository(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database repository: %w", err)
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Add middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	server := &Server{
		config: cfg,
		logger: logger,
		db:     db,
		echo:   e,
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// Start starts the Web UI server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.logger.Info("Starting Web UI server", zap.String("addr", addr))
	return s.echo.Start(addr)
}

// Stop stops the Web UI server
func (s *Server) Stop() error {
	// Close database connection
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return s.echo.Close()
}

// registerRoutes registers the routes for the Web UI server
func (s *Server) registerRoutes() {
	// API routes
	api := s.echo.Group("/api")
	api.GET("/health", s.handleHealth)
	api.GET("/tasks", s.handleGetTasks)
	api.GET("/tasks/:id", s.handleGetTask)
	api.GET("/tasks/:id/logs", s.handleGetTaskLogs)
	api.POST("/tasks", s.handleCreateTask)
	api.GET("/templates", s.handleGetTemplates)
	api.GET("/templates/:id", s.handleGetTemplate)
	api.POST("/templates", s.handleCreateTemplate)

	// Static files
	s.echo.Static("/", "public")
	s.echo.File("/", "public/index.html")
	s.echo.File("/*", "public/index.html")
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetTasks handles requests to get tasks
func (s *Server) handleGetTasks(c echo.Context) error {
	limit := 10
	offset := 0

	tasks, err := s.db.ListTasks(limit, offset)
	if err != nil {
		s.logger.Error("Failed to list tasks", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list tasks"})
	}

	return c.JSON(http.StatusOK, tasks)
}

// handleGetTask handles requests to get a task
func (s *Server) handleGetTask(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Task ID is required"})
	}

	var taskID uint
	if _, err := fmt.Sscanf(id, "%d", &taskID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	task, err := s.db.GetTaskByID(taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Uint("task_id", taskID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get task"})
	}

	if task == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Task not found"})
	}

	return c.JSON(http.StatusOK, task)
}

// handleGetTaskLogs handles requests to get task logs
func (s *Server) handleGetTaskLogs(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Task ID is required"})
	}

	var taskID uint
	if _, err := fmt.Sscanf(id, "%d", &taskID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	logs, err := s.db.GetLogsByTaskID(taskID)
	if err != nil {
		s.logger.Error("Failed to get task logs", zap.Uint("task_id", taskID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get task logs"})
	}

	return c.JSON(http.StatusOK, logs)
}

// handleCreateTask handles requests to create a task
func (s *Server) handleCreateTask(c echo.Context) error {
	var task models.TaskInstance
	if err := c.Bind(&task); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task data"})
	}

	// Set default values
	task.State = models.TaskStatePending
	task.Origin = models.TaskOriginWeb
	task.CreatedBy = 1 // TODO: Get user ID from authentication

	if err := s.db.CreateTask(&task); err != nil {
		s.logger.Error("Failed to create task", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create task"})
	}

	// TODO: Call Core API to create job for task

	return c.JSON(http.StatusCreated, task)
}

// handleGetTemplates handles requests to get templates
func (s *Server) handleGetTemplates(c echo.Context) error {
	templates, err := s.db.ListTemplates()
	if err != nil {
		s.logger.Error("Failed to list templates", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list templates"})
	}

	return c.JSON(http.StatusOK, templates)
}

// handleGetTemplate handles requests to get a template
func (s *Server) handleGetTemplate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Template ID is required"})
	}

	var templateID uint
	if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template ID"})
	}

	template, err := s.db.GetTemplateByID(templateID)
	if err != nil {
		s.logger.Error("Failed to get template", zap.Uint("template_id", templateID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get template"})
	}

	if template == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	return c.JSON(http.StatusOK, template)
}

// handleCreateTemplate handles requests to create a template
func (s *Server) handleCreateTemplate(c echo.Context) error {
	var template models.Template
	if err := c.Bind(&template); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template data"})
	}

	// Set default values
	template.CreatedBy = 1 // TODO: Get user ID from authentication

	if err := s.db.CreateTemplate(&template); err != nil {
		s.logger.Error("Failed to create template", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create template"})
	}

	return c.JSON(http.StatusCreated, template)
}