package database

import (
	"context"
	"errors"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/gorm"
)

// GormReminderRepository is a GORM implementation of ReminderRepository
type GormReminderRepository struct {
	*GormRepository
}

// NewReminderRepository creates a new GormReminderRepository
func NewReminderRepository(db *gorm.DB) ReminderRepository {
	return &GormReminderRepository{
		GormRepository: NewGormRepository(db),
	}
}

// Create creates a new reminder
func (r *GormReminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	if reminder == nil {
		return ErrValidation
	}

	result := r.db.WithContext(ctx).Create(reminder)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetByID gets a reminder by ID
func (r *GormReminderRepository) GetByID(ctx context.Context, id uint) (*models.Reminder, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	var reminder models.Reminder
	result := r.db.WithContext(ctx).First(&reminder, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &reminder, nil
}

// ListByTaskID lists reminders by task ID
func (r *GormReminderRepository) ListByTaskID(ctx context.Context, taskID uint) ([]*models.Reminder, error) {
	if taskID == 0 {
		return nil, ErrInvalidID
	}

	var reminders []*models.Reminder
	result := r.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&reminders)
	if result.Error != nil {
		return nil, result.Error
	}

	return reminders, nil
}

// ListPending lists pending reminders with pagination
func (r *GormReminderRepository) ListPending(ctx context.Context, offset, limit int) ([]*models.Reminder, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var reminders []*models.Reminder
	result := r.db.WithContext(ctx).
		Where("state = ?", models.ReminderStatePending).
		Order("chat_at").
		Offset(offset).
		Limit(limit).
		Find(&reminders)
	if result.Error != nil {
		return nil, result.Error
	}

	return reminders, nil
}

// Update updates a reminder
func (r *GormReminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	if reminder == nil || reminder.ID == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Save(reminder)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete deletes a reminder by ID
func (r *GormReminderRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Delete(&models.Reminder{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
