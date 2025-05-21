package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	// This import would be generated from the proto file
	// pb "github.com/BogdanDolia/ops-butler/api/proto/agent"
)

// Agent represents a cluster agent
type Agent struct {
	config *Config
	logger *zap.Logger
	conn   *grpc.ClientConn
	// client     pb.AgentServiceClient
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
	// a.client = pb.NewAgentServiceClient(conn)

	return nil
}

// register registers the agent with the server
func (a *Agent) register() error {
	a.logger.Info("Registering with server")

	// TODO: Implement registration with the server
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

	// For now, just use the name as the ID
	a.agentID = a.config.Name
	a.logger.Info("Registered with server", zap.String("agent_id", a.agentID))

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
	a.logger.Debug("Sending heartbeat")

	// TODO: Implement heartbeat
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

		// TODO: Implement task execution
		// This would involve:
		// 1. Creating a temporary script file
		// 2. Setting up environment variables from params
		// 3. Executing the script
		// 4. Streaming the output back to the server
		// 5. Updating the task status

		// For now, just simulate a task
		time.Sleep(5 * time.Second)

		// Update the task status
		a.tasksMutex.Lock()
		task.Status = "completed"
		task.EndTime = time.Now()
		task.ExitCode = 0
		a.tasksMutex.Unlock()

		a.logger.Info("Task completed", zap.String("task_id", taskID))
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
