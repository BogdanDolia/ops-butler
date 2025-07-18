package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Agent represents a cluster agent
type Agent struct {
	config     *Config
	logger     *zap.Logger
	conn       *grpc.ClientConn
	agentID    string
	tasks      map[string]*Task
	tasksMutex sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// Task represents a task being executed by the agent
type Task struct {
	ID        string
	Script    string
	Params    map[string]string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int
	Error     string
	Cancel    context.CancelFunc
}

// NewAgent creates a new agent
func NewAgent(config *Config, logger *zap.Logger) *Agent {
	return &Agent{
		config:     config,
		logger:     logger,
		tasks:      make(map[string]*Task),
		tasksMutex: sync.RWMutex{},
		stopCh:     make(chan struct{}),
	}
}

// Start starts the agent
func (a *Agent) Start() error {
	a.logger.Info("Starting agent", zap.String("name", a.config.Name))

	// Connect to the server
	if err := a.connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Register with the server
	if err := a.register(); err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}

	// Start the heartbeat goroutine
	a.wg.Add(1)
	go a.heartbeatLoop()

	return nil
}

// Stop stops the agent
func (a *Agent) Stop() error {
	a.logger.Info("Stopping agent")

	// Signal all goroutines to stop
	close(a.stopCh)

	// Wait for all goroutines to finish
	a.wg.Wait()

	// Cancel all running tasks
	a.tasksMutex.Lock()
	for _, task := range a.tasks {
		if task.Status == "running" && task.Cancel != nil {
			task.Cancel()
		}
	}
	a.tasksMutex.Unlock()

	// Close the connection
	if a.conn != nil {
		if err := a.conn.Close(); err != nil {
			a.logger.Error("Failed to close connection", zap.Error(err))
		}
	}

	a.logger.Info("Agent stopped")
	return nil
}

// connect connects to the server
func (a *Agent) connect() error {
	a.logger.Info("Connecting to server", zap.String("address", a.config.ServerAddress))

	var opts []grpc.DialOption
	if a.config.TLSEnabled {
		creds, err := credentials.NewClientTLSFromFile(a.config.TLSCertFile, "")
		if err != nil {
			return fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(a.config.ServerAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	a.conn = conn

	return nil
}

// register registers the agent with the server
func (a *Agent) register() error {
	a.logger.Info("Registering with server")

	// When the proto code is generated, this would be uncommented and used
	// req := &pb.RegisterRequest{
	//     Name:    a.config.Name,
	//     Labels:  a.config.Labels,
	//     Version: "1.0.0",
	// }
	//
	// resp, err := a.client.Register(context.Background(), req)
	// if err != nil {
	//     return fmt.Errorf("failed to register: %w", err)
	// }
	//
	// if !resp.Success {
	//     return fmt.Errorf("registration failed: %s", resp.Error)
	// }
	//
	// a.agentID = resp.AgentId
	// a.logger.Info("Registered with server", zap.String("agent_id", a.agentID))

	// For now, use the REST API to register
	url := fmt.Sprintf("http://%s/api/v1/agents/register", a.config.ServerAddress)

	// Prepare request body
	reqBody := map[string]interface{}{
		"name":         a.config.Name,
		"cluster_name": a.config.ClusterName,
		"labels":       a.config.Labels,
		"version":      "1.0.0", // Hardcoded for now
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: %s", string(body))
	}

	// Parse response
	var response struct {
		AgentID uint   `json:"agent_id"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Set agent ID
	a.agentID = fmt.Sprintf("%d", response.AgentID)
	a.logger.Info("Registered with server",
		zap.String("agent_id", a.agentID),
		zap.String("cluster_name", a.config.ClusterName),
		zap.Any("labels", a.config.Labels))

	return nil
}

// heartbeatLoop sends heartbeats to the server
func (a *Agent) heartbeatLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.sendHeartbeat(); err != nil {
				a.logger.Error("Failed to send heartbeat", zap.Error(err))
			}
		case <-a.stopCh:
			return
		}
	}
}

// sendHeartbeat sends a heartbeat to the server
func (a *Agent) sendHeartbeat() error {
	a.logger.Debug("Sending heartbeat",
		zap.String("agent_id", a.agentID),
		zap.String("cluster_name", a.config.ClusterName),
		zap.Any("labels", a.config.Labels))

	// When the proto code is generated, this would be uncommented and used
	// req := &pb.HeartbeatRequest{
	//     AgentId: a.agentID,
	//     Labels:  a.config.Labels,
	//     Status:  "healthy",
	// }
	//
	// resp, err := a.client.Heartbeat(context.Background(), req)
	// if err != nil {
	//     return fmt.Errorf("failed to send heartbeat: %w", err)
	// }
	//
	// if !resp.Success {
	//     return fmt.Errorf("heartbeat failed: %s", resp.Error)
	// }

	// For now, use the REST API to send heartbeat
	url := fmt.Sprintf("http://%s/api/v1/agents/heartbeat", a.config.ServerAddress)

	// Prepare request body
	reqBody := map[string]interface{}{
		"agent_id": a.agentID,
		"labels":   a.config.Labels,
		"status":   "healthy",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to send heartbeat to server: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed: %s", string(body))
	}

	a.logger.Info("Heartbeat sent successfully")

	return nil
}

// ExecuteTask executes a task
func (a *Agent) ExecuteTask(taskID, script string, params map[string]string, timeout int32) error {
	a.logger.Info("Executing task", zap.String("task_id", taskID))

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	}

	// Create a new task
	task := &Task{
		ID:        taskID,
		Script:    script,
		Params:    params,
		Status:    "running",
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	// Store the task
	a.tasksMutex.Lock()
	a.tasks[taskID] = task
	a.tasksMutex.Unlock()

	// Execute the task in a goroutine
	go func() {
		defer cancel()

		a.logger.Info("Executing task script",
			zap.String("task_id", taskID),
			zap.String("script", script),
			zap.Any("params", params))

		// Create a temporary directory for the task
		tempDir := fmt.Sprintf("/tmp/ops-butler-task-%s", taskID)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			a.logger.Error("Failed to create temp directory",
				zap.String("task_id", taskID),
				zap.Error(err))

			a.tasksMutex.Lock()
			task.Status = "failed"
			task.EndTime = time.Now()
			task.ExitCode = 1
			task.Error = fmt.Sprintf("Failed to create temp directory: %v", err)
			a.tasksMutex.Unlock()
			return
		}

		// Create the script file
		scriptPath := filepath.Join(tempDir, "script.sh")
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			a.logger.Error("Failed to write script file",
				zap.String("task_id", taskID),
				zap.Error(err))

			a.tasksMutex.Lock()
			task.Status = "failed"
			task.EndTime = time.Now()
			task.ExitCode = 1
			task.Error = fmt.Sprintf("Failed to write script file: %v", err)
			a.tasksMutex.Unlock()
			return
		}

		// Prepare the command
		cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)

		// Set up environment variables from params
		cmd.Env = os.Environ()
		for k, v := range params {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		// Capture stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			a.logger.Error("Failed to create stdout pipe",
				zap.String("task_id", taskID),
				zap.Error(err))

			a.tasksMutex.Lock()
			task.Status = "failed"
			task.EndTime = time.Now()
			task.ExitCode = 1
			task.Error = fmt.Sprintf("Failed to create stdout pipe: %v", err)
			a.tasksMutex.Unlock()
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			a.logger.Error("Failed to create stderr pipe",
				zap.String("task_id", taskID),
				zap.Error(err))

			a.tasksMutex.Lock()
			task.Status = "failed"
			task.EndTime = time.Now()
			task.ExitCode = 1
			task.Error = fmt.Sprintf("Failed to create stderr pipe: %v", err)
			a.tasksMutex.Unlock()
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			a.logger.Error("Failed to start command",
				zap.String("task_id", taskID),
				zap.Error(err))

			a.tasksMutex.Lock()
			task.Status = "failed"
			task.EndTime = time.Now()
			task.ExitCode = 1
			task.Error = fmt.Sprintf("Failed to start command: %v", err)
			a.tasksMutex.Unlock()
			return
		}

		// TODO: Stream output back to the server
		// This would involve reading from stdout and stderr and sending the output
		// to the server using the ExecuteTask streaming RPC.

		// For now, just read the output and log it
		stdoutBytes, _ := io.ReadAll(stdout)
		stderrBytes, _ := io.ReadAll(stderr)

		a.logger.Debug("Task output",
			zap.String("task_id", taskID),
			zap.String("stdout", string(stdoutBytes)),
			zap.String("stderr", string(stderrBytes)))

		// Wait for the command to finish
		err = cmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		// Update the task status
		a.tasksMutex.Lock()
		task.Status = "completed"
		task.EndTime = time.Now()
		task.ExitCode = exitCode
		if exitCode != 0 {
			task.Status = "failed"
			task.Error = fmt.Sprintf("Command exited with code %d", exitCode)
		}
		a.tasksMutex.Unlock()

		a.logger.Info("Task completed",
			zap.String("task_id", taskID),
			zap.Int("exit_code", exitCode))

		// Clean up
		os.RemoveAll(tempDir)
	}()

	return nil
}

// GetTaskStatus gets the status of a task
func (a *Agent) GetTaskStatus(taskID string) (*Task, error) {
	a.tasksMutex.RLock()
	defer a.tasksMutex.RUnlock()

	task, ok := a.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// CancelTask cancels a running task
func (a *Agent) CancelTask(taskID string) error {
	a.tasksMutex.Lock()
	defer a.tasksMutex.Unlock()

	task, ok := a.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != "running" {
		return fmt.Errorf("task is not running: %s", taskID)
	}

	if task.Cancel != nil {
		task.Cancel()
	}

	task.Status = "cancelled"
	task.EndTime = time.Now()
	task.Error = "Task cancelled by user"

	a.logger.Info("Task cancelled", zap.String("task_id", taskID))

	return nil
}
