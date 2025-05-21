package database

import (
	"context"
	"errors"
	"time"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/gorm"
)

// GormTaskRepository is a GORM implementation of TaskRepository
type GormTaskRepository struct {
	*GormRepository
}

// NewTaskRepository creates a new GormTaskRepository
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &GormTaskRepository{
		GormRepository: NewGormRepository(db),
	}
}

// Create creates a new task
func (r *GormTaskRepository) Create(ctx context.Context, task *models.TaskInstance) error {
	if task == nil {
		return ErrValidation
	}

	result := r.db.WithContext(ctx).Create(task)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetByID gets a task by ID
func (r *GormTaskRepository) GetByID(ctx context.Context, id uint) (*models.TaskInstance, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	var task models.TaskInstance
	result := r.db.WithContext(ctx).First(&task, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &task, nil
}

// List lists tasks with pagination
func (r *GormTaskRepository) List(ctx context.Context, offset, limit int) ([]*models.TaskInstance, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var tasks []*models.TaskInstance
	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// ListByTemplateID lists tasks by template ID with pagination
func (r *GormTaskRepository) ListByTemplateID(ctx context.Context, templateID uint, offset, limit int) ([]*models.TaskInstance, error) {
	if templateID == 0 {
		return nil, ErrInvalidID
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var tasks []*models.TaskInstance
	result := r.db.WithContext(ctx).Where("template_id = ?", templateID).Offset(offset).Limit(limit).Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// ListByState lists tasks by state with pagination
func (r *GormTaskRepository) ListByState(ctx context.Context, state models.TaskState, offset, limit int) ([]*models.TaskInstance, error) {
	if state == "" {
		return nil, ErrValidation
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var tasks []*models.TaskInstance
	result := r.db.WithContext(ctx).Where("state = ?", state).Offset(offset).Limit(limit).Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// ListDue lists tasks that are due with pagination
func (r *GormTaskRepository) ListDue(ctx context.Context, offset, limit int) ([]*models.TaskInstance, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var tasks []*models.TaskInstance
	result := r.db.WithContext(ctx).
		Where("state = ? AND due_at <= ?", models.TaskStatePending, time.Now()).
		Offset(offset).
		Limit(limit).
		Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// Update updates a task
func (r *GormTaskRepository) Update(ctx context.Context, task *models.TaskInstance) error {
	if task == nil || task.ID == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete deletes a task by ID
func (r *GormTaskRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Delete(&models.TaskInstance{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
