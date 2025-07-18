package database

import (
	"context"
	"errors"

	"github.com/BogdanDolia/ops-butler/internal/models"
	"gorm.io/gorm"
)

// GormAgentRepository is a GORM implementation of AgentRepository
type GormAgentRepository struct {
	*GormRepository
}

// NewAgentRepository creates a new GormAgentRepository
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &GormAgentRepository{
		GormRepository: NewGormRepository(db),
	}
}

// Create creates a new agent
func (r *GormAgentRepository) Create(ctx context.Context, agent *models.ClusterAgent) error {
	if agent == nil {
		return ErrValidation
	}

	result := r.db.WithContext(ctx).Create(agent)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// GetByID gets an agent by ID
func (r *GormAgentRepository) GetByID(ctx context.Context, id uint) (*models.ClusterAgent, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	var agent models.ClusterAgent
	result := r.db.WithContext(ctx).First(&agent, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &agent, nil
}

// GetByName gets an agent by name
func (r *GormAgentRepository) GetByName(ctx context.Context, name string) (*models.ClusterAgent, error) {
	if name == "" {
		return nil, ErrValidation
	}

	var agent models.ClusterAgent
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&agent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}

	return &agent, nil
}

// List lists agents with pagination
func (r *GormAgentRepository) List(ctx context.Context, offset, limit int) ([]*models.ClusterAgent, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	var agents []*models.ClusterAgent
	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}

	return agents, nil
}

// Update updates an agent
func (r *GormAgentRepository) Update(ctx context.Context, agent *models.ClusterAgent) error {
	if agent == nil || agent.ID == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Save(agent)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete deletes an agent by ID
func (r *GormAgentRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	result := r.db.WithContext(ctx).Delete(&models.ClusterAgent{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
