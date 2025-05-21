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

	"github.com/BogdanDolia/ops-butler/internal/config"
	"github.com/BogdanDolia/ops-butler/internal/database"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	config     *config.Config
	logger     *zap.Logger
	db         *database.GormRepository
	templates  database.TemplateRepository
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
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Create task"})
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
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Execute task"})
}

func (s *Server) handleGetTaskLogs(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Get task logs"})
}

func (s *Server) handleListAgents(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "List agents"})
}

func (s *Server) handleGetAgent(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusOK, gin.H{"message": "Get agent"})
}

func (s *Server) handleWebSocketLogs(c *gin.Context) {
	// TODO: Implement WebSocket handler
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket logs"})
}
