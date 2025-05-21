package database

import (
	"context"
	"errors"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrNotFound is returned when a record is not found
	ErrNotFound = errors.New("record not found")
	// ErrInvalidID is returned when an invalid ID is provided
	ErrInvalidID = errors.New("invalid ID")
	// ErrValidation is returned when validation fails
	ErrValidation = errors.New("validation failed")
)

// Repository is the interface that all repositories must implement
type Repository interface {
	Close() error
}

// TemplateRepository is the interface for template operations
type TemplateRepository interface {
	Repository
	Create(ctx context.Context, template *models.Template) error
	GetByID(ctx context.Context, id uint) (*models.Template, error)
	GetByName(ctx context.Context, name string) (*models.Template, error)
	List(ctx context.Context, offset, limit int) ([]*models.Template, error)
	Update(ctx context.Context, template *models.Template) error
	Delete(ctx context.Context, id uint) error
}

// TaskRepository is the interface for task operations
type TaskRepository interface {
	Repository
	Create(ctx context.Context, task *models.TaskInstance) error
	GetByID(ctx context.Context, id uint) (*models.TaskInstance, error)
	List(ctx context.Context, offset, limit int) ([]*models.TaskInstance, error)
	ListByTemplateID(ctx context.Context, templateID uint, offset, limit int) ([]*models.TaskInstance, error)
	ListByState(ctx context.Context, state models.TaskState, offset, limit int) ([]*models.TaskInstance, error)
	ListDue(ctx context.Context, offset, limit int) ([]*models.TaskInstance, error)
	Update(ctx context.Context, task *models.TaskInstance) error
	Delete(ctx context.Context, id uint) error
}

// ReminderRepository is the interface for reminder operations
type ReminderRepository interface {
	Repository
	Create(ctx context.Context, reminder *models.Reminder) error
	GetByID(ctx context.Context, id uint) (*models.Reminder, error)
	ListByTaskID(ctx context.Context, taskID uint) ([]*models.Reminder, error)
	ListPending(ctx context.Context, offset, limit int) ([]*models.Reminder, error)
	Update(ctx context.Context, reminder *models.Reminder) error
	Delete(ctx context.Context, id uint) error
}

// ExecutionLogRepository is the interface for execution log operations
type ExecutionLogRepository interface {
	Repository
	Create(ctx context.Context, log *models.ExecutionLog) error
	GetByID(ctx context.Context, id uint) (*models.ExecutionLog, error)
	ListByTaskID(ctx context.Context, taskID uint, offset, limit int) ([]*models.ExecutionLog, error)
	ListByAgentID(ctx context.Context, agentID uint, offset, limit int) ([]*models.ExecutionLog, error)
}

// AgentRepository is the interface for agent operations
type AgentRepository interface {
	Repository
	Create(ctx context.Context, agent *models.ClusterAgent) error
	GetByID(ctx context.Context, id uint) (*models.ClusterAgent, error)
	GetByName(ctx context.Context, name string) (*models.ClusterAgent, error)
	List(ctx context.Context, offset, limit int) ([]*models.ClusterAgent, error)
	Update(ctx context.Context, agent *models.ClusterAgent) error
	Delete(ctx context.Context, id uint) error
}

// UserRepository is the interface for user operations
type UserRepository interface {
	Repository
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByExternalID(ctx context.Context, externalID string) (*models.User, error)
	List(ctx context.Context, offset, limit int) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
}

// GormRepository is a base repository implementation using GORM
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository creates a new GormRepository
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// DB returns the underlying database connection
func (r *GormRepository) DB() *gorm.DB {
	return r.db
}

// Close closes the database connection
func (r *GormRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
