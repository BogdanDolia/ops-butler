package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/models"
	slackClient "github.com/BogdanDolia/ops-butler/new-ops-butler/internal/slack"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// Config represents the configuration for the Core server
type Config struct {
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
	Database *database.Config    `json:"database"`
	Slack    *slackClient.Config `json:"slack"`
	K8s      *K8sConfig          `json:"k8s"`
}

// Server represents the Core server
type Server struct {
	config     *Config
	logger     *zap.Logger
	db         database.Repository
	slack      *slackClient.Client
	k8s        *K8sClient
	httpServer *http.Server
}

// NewServer creates a new Core server
func NewServer(cfg *Config, logger *zap.Logger) (*Server, error) {
	// Create database repository
	db, err := database.NewRepository(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database repository: %w", err)
	}

	// Create Slack client
	slack, err := slackClient.NewClient(cfg.Slack, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Slack client: %w", err)
	}

	// Create Kubernetes client
	k8s, err := NewK8sClient(cfg.K8s, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &Server{
		config: cfg,
		logger: logger,
		db:     db,
		slack:  slack,
		k8s:    k8s,
	}, nil
}

// Start starts the Core server
func (s *Server) Start() error {
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/slack/command", s.handleSlackCommand)
	mux.HandleFunc("/slack/event", s.handleSlackEvent)

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start background task processor
	go s.processTasksLoop()

	// Start HTTP server
	s.logger.Info("Starting Core server", zap.String("addr", addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop stops the Core server
func (s *Server) Stop() error {
	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleSlackCommand handles Slack slash commands
func (s *Server) handleSlackCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify Slack request
	verifier, err := slack.NewSecretsVerifier(r.Header, s.config.Slack.SigningKey)
	if err != nil {
		s.logger.Error("Failed to create secrets verifier", zap.Error(err))
		http.Error(w, "Failed to verify request", http.StatusBadRequest)
		return
	}

	// Read and verify request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Add body to verifier
	verifier.Write(body)
	if err := verifier.Ensure(); err != nil {
		s.logger.Error("Failed to verify request signature", zap.Error(err))
		http.Error(w, "Invalid request signature", http.StatusUnauthorized)
		return
	}

	// Parse form values from the body we already read
	values, err := url.ParseQuery(string(body))
	if err != nil {
		s.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract command parameters
	command := values.Get("command")
	text := values.Get("text")
	channelID := values.Get("channel_id")
	userID := values.Get("user_id")
	userName := values.Get("user_name")

	// Parse command
	subCommand, args, err := s.slack.ParseSlashCommand(command, text)
	if err != nil {
		s.logger.Error("Failed to parse slash command",
			zap.String("command", command),
			zap.String("text", text),
			zap.Error(err),
		)
		http.Error(w, fmt.Sprintf("Failed to parse command: %v", err), http.StatusBadRequest)
		return
	}

	// Handle command
	var response string
	switch subCommand {
	case "help":
		response = s.handleHelpCommand()
	case "status":
		response = s.handleStatusCommand(channelID, userID, userName)
	case "collect-logs":
		response = s.handleCollectLogsCommand(channelID, userID, userName, args)
	case "list":
		response = s.handleListCommand(channelID, userID, userName)
	default:
		response = fmt.Sprintf("Unknown command: %s. Type `/ops help` for available commands.", subCommand)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"response_type": "ephemeral", "text": response})
}

// handleSlackEvent handles Slack events
func (s *Server) handleSlackEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify Slack request
	verifier, err := slack.NewSecretsVerifier(r.Header, s.config.Slack.SigningKey)
	if err != nil {
		s.logger.Error("Failed to create secrets verifier", zap.Error(err))
		http.Error(w, "Failed to verify request", http.StatusBadRequest)
		return
	}

	// Read and verify request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Add body to verifier
	verifier.Write(body)
	if err := verifier.Ensure(); err != nil {
		s.logger.Error("Failed to verify request signature", zap.Error(err))
		http.Error(w, "Invalid request signature", http.StatusUnauthorized)
		return
	}

	// Parse event
	var event struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		s.logger.Error("Failed to parse event", zap.Error(err))
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	// Handle URL verification
	if event.Type == "url_verification" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(event.Challenge))
		return
	}

	// Acknowledge event
	w.WriteHeader(http.StatusOK)
}

// handleHelpCommand handles the help command
func (s *Server) handleHelpCommand() string {
	return `Available commands:
- /ops help - Show this help message
- /ops status - Show system status
- /ops collect-logs [pod-name] [namespace] - Collect logs from a pod (if namespace is not provided, searches in all namespaces)
- /ops list - List recent tasks`
}

// handleStatusCommand handles the status command
func (s *Server) handleStatusCommand(channelID, userID, userName string) string {
	// Create a task for status check
	task := &models.TaskInstance{
		TaskType:     models.TaskTypeStatus,
		State:        models.TaskStatePending,
		Origin:       models.TaskOriginSlack,
		SlackChannel: channelID,
		CreatedBy:    1, // TODO: Get user ID from database
	}

	if err := s.db.CreateTask(task); err != nil {
		s.logger.Error("Failed to create task",
			zap.String("task_type", string(task.TaskType)),
			zap.Error(err),
		)
		return "Failed to create status check task. Please try again later."
	}

	// Create Kubernetes job for the task
	if err := s.k8s.CreateJobForTask(task); err != nil {
		s.logger.Error("Failed to create job for task",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
		return "Failed to create status check job. Please try again later."
	}

	// Update task in database
	if err := s.db.UpdateTask(task); err != nil {
		s.logger.Error("Failed to update task",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
	}

	return fmt.Sprintf("Status check task created (ID: %d). You'll receive the results shortly.", task.ID)
}

// handleCollectLogsCommand handles the collect-logs command
func (s *Server) handleCollectLogsCommand(channelID, userID, userName string, args []string) string {
	if len(args) < 1 {
		return "Please specify a pod name. Usage: `/ops collect-logs [pod-name] [namespace]`"
	}

	podName := args[0]

	// Check if namespace is provided
	var namespace string
	if len(args) > 1 {
		namespace = args[1]
	}

	// Create a task for log collection
	task := &models.TaskInstance{
		TaskType:     models.TaskTypeCollectLogs,
		State:        models.TaskStatePending,
		Origin:       models.TaskOriginSlack,
		SlackChannel: channelID,
		CreatedBy:    1, // TODO: Get user ID from database
		Params: models.JSONSchema{
			"pod_name": podName,
		},
	}

	// Add namespace to parameters if provided
	if namespace != "" {
		task.Params["namespace"] = namespace
	}

	if err := s.db.CreateTask(task); err != nil {
		s.logger.Error("Failed to create task",
			zap.String("task_type", string(task.TaskType)),
			zap.Error(err),
		)
		return "Failed to create log collection task. Please try again later."
	}

	// Create Kubernetes job for the task
	if err := s.k8s.CreateJobForTask(task); err != nil {
		s.logger.Error("Failed to create job for task",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
		return "Failed to create log collection job. Please try again later."
	}

	// Update task in database
	if err := s.db.UpdateTask(task); err != nil {
		s.logger.Error("Failed to update task",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
	}

	return fmt.Sprintf("Log collection task created for pod '%s' (ID: %d). You'll receive the logs shortly.", podName, task.ID)
}

// handleListCommand handles the list command
func (s *Server) handleListCommand(channelID, userID, userName string) string {
	// Get recent tasks
	tasks, err := s.db.ListTasks(10, 0)
	if err != nil {
		s.logger.Error("Failed to list tasks", zap.Error(err))
		return "Failed to list tasks. Please try again later."
	}

	if len(tasks) == 0 {
		return "No tasks found."
	}

	// Format task list
	var sb strings.Builder
	sb.WriteString("Recent tasks:\n")
	for _, task := range tasks {
		sb.WriteString(fmt.Sprintf("- ID: %d, Type: %s, Status: %s\n",
			task.ID, task.TaskType, task.State))
	}

	return sb.String()
}

// processTasksLoop processes tasks in a loop
func (s *Server) processTasksLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processTasks()
		}
	}
}

// processTasks processes tasks
func (s *Server) processTasks() {
	// Get running tasks
	tasks, err := s.db.ListTasksByState(models.TaskStateScheduled, 100, 0)
	if err != nil {
		s.logger.Error("Failed to list scheduled tasks", zap.Error(err))
		return
	}

	for _, task := range tasks {
		// Skip tasks without a job name
		if task.JobName == "" {
			continue
		}

		// Get job status
		state, exitCode, err := s.k8s.GetJobStatus(task.JobName)
		if err != nil {
			s.logger.Error("Failed to get job status",
				zap.Uint("task_id", task.ID),
				zap.String("job_name", task.JobName),
				zap.Error(err),
			)
			continue
		}

		// If task state has changed, update it
		if state != task.State {
			task.State = state
			if state == models.TaskStateCompleted || state == models.TaskStateFailed {
				task.CompletedAt = &time.Time{}
				*task.CompletedAt = time.Now()
				task.ExitCode = &exitCode

				// Get job logs
				logs, err := s.k8s.GetJobLogs(task.JobName)
				if err != nil {
					s.logger.Error("Failed to get job logs",
						zap.Uint("task_id", task.ID),
						zap.String("job_name", task.JobName),
						zap.Error(err),
					)
				} else {
					// Create log entry
					logEntry := &models.ExecutionLog{
						TaskID:    task.ID,
						Chunk:     logs,
						Timestamp: time.Now(),
						Stream:    "stdout",
						Sequence:  0,
					}
					if err := s.db.CreateLog(logEntry); err != nil {
						s.logger.Error("Failed to create log entry",
							zap.Uint("task_id", task.ID),
							zap.Error(err),
						)
					}

					// Send task update to Slack
					message := fmt.Sprintf("Task %d (%s) %s", task.ID, task.TaskType, state)
					if state == models.TaskStateCompleted {
						message += "\n```\n" + logs + "\n```"
					} else {
						message += "\nError: " + logs
					}
					if err := s.slack.SendTaskUpdate(&task, message); err != nil {
						s.logger.Error("Failed to send task update to Slack",
							zap.Uint("task_id", task.ID),
							zap.Error(err),
						)
					}
				}
			}

			// Update task in database
			if err := s.db.UpdateTask(&task); err != nil {
				s.logger.Error("Failed to update task",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
				)
			}
		}
	}

	// Get running tasks
	tasks, err = s.db.ListTasksByState(models.TaskStateRunning, 100, 0)
	if err != nil {
		s.logger.Error("Failed to list running tasks", zap.Error(err))
		return
	}

	for _, task := range tasks {
		// Skip tasks without a job name
		if task.JobName == "" {
			continue
		}

		// Get job status
		state, exitCode, err := s.k8s.GetJobStatus(task.JobName)
		if err != nil {
			s.logger.Error("Failed to get job status",
				zap.Uint("task_id", task.ID),
				zap.String("job_name", task.JobName),
				zap.Error(err),
			)
			continue
		}

		// If task state has changed, update it
		if state != task.State {
			task.State = state
			if state == models.TaskStateCompleted || state == models.TaskStateFailed {
				task.CompletedAt = &time.Time{}
				*task.CompletedAt = time.Now()
				task.ExitCode = &exitCode

				// Get job logs
				logs, err := s.k8s.GetJobLogs(task.JobName)
				if err != nil {
					s.logger.Error("Failed to get job logs",
						zap.Uint("task_id", task.ID),
						zap.String("job_name", task.JobName),
						zap.Error(err),
					)
				} else {
					// Create log entry
					logEntry := &models.ExecutionLog{
						TaskID:    task.ID,
						Chunk:     logs,
						Timestamp: time.Now(),
						Stream:    "stdout",
						Sequence:  0,
					}
					if err := s.db.CreateLog(logEntry); err != nil {
						s.logger.Error("Failed to create log entry",
							zap.Uint("task_id", task.ID),
							zap.Error(err),
						)
					}

					// Send task update to Slack
					message := fmt.Sprintf("Task %d (%s) %s", task.ID, task.TaskType, state)
					if state == models.TaskStateCompleted {
						message += "\n```\n" + logs + "\n```"
					} else {
						message += "\nError: " + logs
					}
					if err := s.slack.SendTaskUpdate(&task, message); err != nil {
						s.logger.Error("Failed to send task update to Slack",
							zap.Uint("task_id", task.ID),
							zap.Error(err),
						)
					}
				}
			}

			// Update task in database
			if err := s.db.UpdateTask(&task); err != nil {
				s.logger.Error("Failed to update task",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
				)
			}
		}
	}
}
