package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSONSchema represents a JSON schema for template parameters
type JSONSchema map[string]interface{}

// Scan implements the sql.Scanner interface for JSONSchema
func (j *JSONSchema) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONSchema value")
	}

	result := make(map[string]interface{})
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

// Value implements the driver.Valuer interface for JSONSchema
func (j JSONSchema) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// TaskState represents the state of a task instance
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateScheduled TaskState = "scheduled"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// TaskOrigin represents the origin of a task instance
type TaskOrigin string

const (
	TaskOriginWeb       TaskOrigin = "web"
	TaskOriginSlack     TaskOrigin = "slack"
	TaskOriginAPI       TaskOrigin = "api"
	TaskOriginScheduler TaskOrigin = "scheduler"
)

// TaskType represents the type of a task
type TaskType string

const (
	TaskTypeCollectLogs TaskType = "collect_logs"
	TaskTypeStatus      TaskType = "status"
	TaskTypeCustom      TaskType = "custom"
	// Add other task types as needed
)

// Template represents a task template that wraps a script with parameter schema
type Template struct {
	gorm.Model
	Name            string         `json:"name" gorm:"uniqueIndex"`
	Description     string         `json:"description"`
	Script          string         `json:"script"`
	ParamsSchema    JSONSchema     `json:"params_schema" gorm:"type:json"`
	RequireApproval bool           `json:"require_approval" gorm:"default:false"`
	CreatedBy       uint           `json:"created_by"`
	TaskInstances   []TaskInstance `json:"-" gorm:"foreignKey:TemplateID"`
}

// TaskInstance represents an instance of a task to be executed
type TaskInstance struct {
	gorm.Model
	TemplateID   *uint          `json:"template_id" gorm:"index"`
	Template     *Template      `json:"-" gorm:"foreignKey:TemplateID"`
	TaskType     TaskType       `json:"task_type" gorm:"default:'collect_logs'"`
	Params       JSONSchema     `json:"params" gorm:"type:json"`
	State        TaskState      `json:"state" gorm:"default:'pending'"`
	DueAt        *time.Time     `json:"due_at"`
	Origin       TaskOrigin     `json:"origin"`
	SlackChannel string         `json:"slack_channel"`
	SlackThread  string         `json:"slack_thread"`
	CreatedBy    uint           `json:"created_by"`
	JobName      string         `json:"job_name"`
	Logs         []ExecutionLog `json:"-" gorm:"foreignKey:TaskID"`
	ApprovedBy   *uint          `json:"approved_by"`
	ApprovedAt   *time.Time     `json:"approved_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	ExitCode     *int           `json:"exit_code"`
}

// ExecutionLog represents a log chunk from task execution
type ExecutionLog struct {
	gorm.Model
	TaskID    uint         `json:"task_id" gorm:"index"`
	Task      TaskInstance `json:"-" gorm:"foreignKey:TaskID"`
	Chunk     string       `json:"chunk"`
	Timestamp time.Time    `json:"timestamp" gorm:"index"`
	Stream    string       `json:"stream"` // stdout, stderr
	Sequence  int          `json:"sequence" gorm:"index"`
}

// User represents a user in the system
type User struct {
	gorm.Model
	Email       string     `json:"email" gorm:"uniqueIndex"`
	Name        string     `json:"name"`
	Role        string     `json:"role" gorm:"default:'viewer'"`
	SlackUserID string     `json:"slack_user_id" gorm:"uniqueIndex"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// Config represents system configuration
type Config struct {
	gorm.Model
	Key   string `json:"key" gorm:"uniqueIndex"`
	Value string `json:"value"`
}