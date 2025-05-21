package database

import (
	"context"
	"errors"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/gorm"
)

// GormTemplateRepository is a GORM implementation of TemplateRepository
type GormTemplateRepository struct {
	*GormRepository
}

// NewTemplateRepository creates a new GormTemplateRepository
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &GormTemplateRepository{
		GormRepository: NewGormRepository(db),
	}
}

// Create creates a new template
func (r *GormTemplateRepository) Create(ctx context.Context, template *models.Template) error {
	if template == nil {
		return ErrValidation
	}

	result := r.db.WithContext(ctx).Create(template)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetByID gets a template by ID
func (r *GormTemplateRepository) GetByID(ctx context.Context, id uint) (*models.Template, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	var template models.Template
	result := r.db.WithContext(ctx).First(&template, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &template, nil
}

// GetByName gets a template by name
func (r *GormTemplateRepository) GetByName(ctx context.Context, name string) (*models.Template, error) {
	if name == "" {
		return nil, ErrValidation
	}

	var template models.Template
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&template)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &template, nil
}

// List lists templates with pagination
func (r *GormTemplateRepository) List(ctx context.Context, offset, limit int) ([]*models.Template, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var templates []*models.Template
	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}

	return templates, nil
}

// Update updates a template
func (r *GormTemplateRepository) Update(ctx context.Context, template *models.Template) error {
	if template == nil || template.ID == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Save(template)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete deletes a template by ID
func (r *GormTemplateRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Delete(&models.Template{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
