package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Template represents a task template that wraps a script with parameter schema
type Template struct {
	gorm.Model
	Name            string         `json:"name" gorm:"uniqueIndex"`
	Description     string         `json:"description"`
	Script          string         `json:"script"`
	ParamsSchema    JSONSchema     `json:"params_schema" gorm:"type:jsonb"`
	RequireApproval bool           `json:"require_approval" gorm:"default:false"`
	CreatedBy       uint           `json:"created_by"`
	TaskInstances   []TaskInstance `json:"-" gorm:"foreignKey:TemplateID"`
}

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
	TaskOriginWeb        TaskOrigin = "web"
	TaskOriginSlack      TaskOrigin = "slack"
	TaskOriginGoogleChat TaskOrigin = "google_chat"
	TaskOriginAPI        TaskOrigin = "api"
	TaskOriginScheduler  TaskOrigin = "scheduler"
	TaskOriginSheet      TaskOrigin = "sheet"
)

// TaskInstance represents an instance of a task to be executed
type TaskInstance struct {
	gorm.Model
	TemplateID  uint           `json:"template_id" gorm:"index"`
	Template    Template       `json:"-" gorm:"foreignKey:TemplateID"`
	Params      JSONSchema     `json:"params" gorm:"type:jsonb"`
	State       TaskState      `json:"state" gorm:"default:'pending'"`
	DueAt       *time.Time     `json:"due_at"`
	Origin      TaskOrigin     `json:"origin"`
	ChatThread  string         `json:"chat_thread"`
	CreatedBy   uint           `json:"created_by"`
	ExecutedBy  *uint          `json:"executed_by"`
	AgentID     *uint          `json:"agent_id"`
	Agent       *ClusterAgent  `json:"-" gorm:"foreignKey:AgentID"`
	Reminders   []Reminder     `json:"-" gorm:"foreignKey:TaskID"`
	Logs        []ExecutionLog `json:"-" gorm:"foreignKey:TaskID"`
	ApprovedBy  *uint          `json:"approved_by"`
	ApprovedAt  *time.Time     `json:"approved_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	ExitCode    *int           `json:"exit_code"`
}

// ReminderState represents the state of a reminder
type ReminderState string

const (
	ReminderStatePending   ReminderState = "pending"
	ReminderStateDelivered ReminderState = "delivered"
	ReminderStateActioned  ReminderState = "actioned"
	ReminderStateCancelled ReminderState = "cancelled"
)

// Reminder represents a scheduled reminder for a task
type Reminder struct {
	gorm.Model
	TaskID     uint          `json:"task_id" gorm:"index"`
	Task       TaskInstance  `json:"-" gorm:"foreignKey:TaskID"`
	ChatAt     time.Time     `json:"chat_at" gorm:"index"`
	State      ReminderState `json:"state" gorm:"default:'pending'"`
	ChatType   string        `json:"chat_type"` // slack, google_chat
	ChatID     string        `json:"chat_id"`   // channel ID, space name, etc.
	MessageID  string        `json:"message_id"`
	SnoozedAt  *time.Time    `json:"snoozed_at"`
	SnoozedBy  *uint         `json:"snoozed_by"`
	CancelledAt *time.Time   `json:"cancelled_at"`
	CancelledBy *uint        `json:"cancelled_by"`
}

// ExecutionLog represents a log chunk from task execution
type ExecutionLog struct {
	gorm.Model
	TaskID    uint      `json:"task_id" gorm:"index"`
	Task      TaskInstance `json:"-" gorm:"foreignKey:TaskID"`
	AgentID   uint      `json:"agent_id" gorm:"index"`
	Agent     ClusterAgent `json:"-" gorm:"foreignKey:AgentID"`
	Chunk     string    `json:"chunk"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
	Stream    string    `json:"stream"` // stdout, stderr
	Sequence  int       `json:"sequence" gorm:"index"`
}

// ClusterAgent represents a cluster agent
type ClusterAgent struct {
	gorm.Model
	Name          string            `json:"name" gorm:"uniqueIndex"`
	Labels        JSONSchema        `json:"labels" gorm:"type:jsonb"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Status        string            `json:"status" gorm:"default:'unknown'"`
	Version       string            `json:"version"`
	TaskInstances []TaskInstance    `json:"-" gorm:"foreignKey:AgentID"`
	Logs          []ExecutionLog    `json:"-" gorm:"foreignKey:AgentID"`
}

// User represents a user in the system
type User struct {
	gorm.Model
	Email        string `json:"email" gorm:"uniqueIndex"`
	Name         string `json:"name"`
	Role         string `json:"role" gorm:"default:'viewer'"`
	ExternalID   string `json:"external_id" gorm:"uniqueIndex"`
	Provider     string `json:"provider"` // github, google, etc.
	LastLoginAt  *time.Time `json:"last_login_at"`
}