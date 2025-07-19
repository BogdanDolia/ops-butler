package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Config represents database configuration
type Config struct {
	Type     string `json:"type"`     // sqlite, postgres, mysql
	Host     string `json:"host"`     // for postgres, mysql
	Port     int    `json:"port"`     // for postgres, mysql
	User     string `json:"user"`     // for postgres, mysql
	Password string `json:"password"` // for postgres, mysql
	DBName   string `json:"dbname"`   // for postgres, mysql, sqlite
	SSLMode  string `json:"sslmode"`  // for postgres
	Path     string `json:"path"`     // for sqlite
}

// Repository defines the interface for database operations
type Repository interface {
	// Template operations
	CreateTemplate(template *models.Template) error
	GetTemplateByID(id uint) (*models.Template, error)
	GetTemplateByName(name string) (*models.Template, error)
	ListTemplates() ([]models.Template, error)
	UpdateTemplate(template *models.Template) error
	DeleteTemplate(id uint) error

	// TaskInstance operations
	CreateTask(task *models.TaskInstance) error
	GetTaskByID(id uint) (*models.TaskInstance, error)
	ListTasks(limit, offset int) ([]models.TaskInstance, error)
	ListTasksByState(state models.TaskState, limit, offset int) ([]models.TaskInstance, error)
	UpdateTask(task *models.TaskInstance) error
	DeleteTask(id uint) error

	// ExecutionLog operations
	CreateLog(log *models.ExecutionLog) error
	GetLogsByTaskID(taskID uint) ([]models.ExecutionLog, error)

	// User operations
	CreateUser(user *models.User) error
	GetUserByID(id uint) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserBySlackID(slackID string) (*models.User, error)
	ListUsers() ([]models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error

	// Config operations
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error

	// DB operations
	Close() error
}

// GormRepository implements Repository using GORM
type GormRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewRepository creates a new repository based on the configuration
func NewRepository(cfg *Config, logger *zap.Logger) (Repository, error) {
	var db *gorm.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		// Ensure directory exists
		if cfg.Path != "" {
			dir := filepath.Dir(cfg.Path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory for SQLite database: %w", err)
			}
		}

		// Default to in-memory if no path is provided
		dbPath := cfg.Path
		if dbPath == "" {
			dbPath = "file::memory:?cache=shared"
		}

		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
		}

	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
		}

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
		}

	default:
		return nil, errors.New("unsupported database type")
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return &GormRepository{
		db:     db,
		logger: logger,
	}, nil
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Template{},
		&models.TaskInstance{},
		&models.ExecutionLog{},
		&models.User{},
		&models.Config{},
	)
}

// Close closes the database connection
func (r *GormRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Template operations

// CreateTemplate creates a new template
func (r *GormRepository) CreateTemplate(template *models.Template) error {
	return r.db.Create(template).Error
}

// GetTemplateByID gets a template by ID
func (r *GormRepository) GetTemplateByID(id uint) (*models.Template, error) {
	var template models.Template
	if err := r.db.First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// GetTemplateByName gets a template by name
func (r *GormRepository) GetTemplateByName(name string) (*models.Template, error) {
	var template models.Template
	if err := r.db.Where("name = ?", name).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// ListTemplates lists all templates
func (r *GormRepository) ListTemplates() ([]models.Template, error) {
	var templates []models.Template
	if err := r.db.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// UpdateTemplate updates a template
func (r *GormRepository) UpdateTemplate(template *models.Template) error {
	return r.db.Save(template).Error
}

// DeleteTemplate deletes a template
func (r *GormRepository) DeleteTemplate(id uint) error {
	return r.db.Delete(&models.Template{}, id).Error
}

// TaskInstance operations

// CreateTask creates a new task
func (r *GormRepository) CreateTask(task *models.TaskInstance) error {
	return r.db.Create(task).Error
}

// GetTaskByID gets a task by ID
func (r *GormRepository) GetTaskByID(id uint) (*models.TaskInstance, error) {
	var task models.TaskInstance
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks lists all tasks with pagination
func (r *GormRepository) ListTasks(limit, offset int) ([]models.TaskInstance, error) {
	var tasks []models.TaskInstance
	if err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListTasksByState lists tasks by state with pagination
func (r *GormRepository) ListTasksByState(state models.TaskState, limit, offset int) ([]models.TaskInstance, error) {
	var tasks []models.TaskInstance
	if err := r.db.Where("state = ?", state).Limit(limit).Offset(offset).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTask updates a task
func (r *GormRepository) UpdateTask(task *models.TaskInstance) error {
	return r.db.Save(task).Error
}

// DeleteTask deletes a task
func (r *GormRepository) DeleteTask(id uint) error {
	return r.db.Delete(&models.TaskInstance{}, id).Error
}

// ExecutionLog operations

// CreateLog creates a new execution log
func (r *GormRepository) CreateLog(log *models.ExecutionLog) error {
	return r.db.Create(log).Error
}

// GetLogsByTaskID gets logs by task ID
func (r *GormRepository) GetLogsByTaskID(taskID uint) ([]models.ExecutionLog, error) {
	var logs []models.ExecutionLog
	if err := r.db.Where("task_id = ?", taskID).Order("sequence ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// User operations

// CreateUser creates a new user
func (r *GormRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

// GetUserByID gets a user by ID
func (r *GormRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail gets a user by email
func (r *GormRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserBySlackID gets a user by Slack ID
func (r *GormRepository) GetUserBySlackID(slackID string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("slack_user_id = ?", slackID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers lists all users
func (r *GormRepository) ListUsers() ([]models.User, error) {
	var users []models.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUser updates a user
func (r *GormRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

// DeleteUser deletes a user
func (r *GormRepository) DeleteUser(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// Config operations

// GetConfig gets a configuration value by key
func (r *GormRepository) GetConfig(key string) (string, error) {
	var config models.Config
	if err := r.db.Where("key = ?", key).First(&config).Error; err != nil {
		return "", err
	}
	return config.Value, nil
}

// SetConfig sets a configuration value
func (r *GormRepository) SetConfig(key, value string) error {
	var config models.Config
	result := r.db.Where("key = ?", key).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			config = models.Config{
				Key:   key,
				Value: value,
			}
			return r.db.Create(&config).Error
		}
		return result.Error
	}
	config.Value = value
	return r.db.Save(&config).Error
}
