package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/BogdanDolia/ops-butler/internal/database"
	"github.com/BogdanDolia/ops-butler/internal/models"
)

// Scheduler represents a task scheduler
type Scheduler struct {
	config    *Config
	logger    *zap.Logger
	db        *gorm.DB
	redis     *redis.Client
	tasks     database.TaskRepository
	reminders database.ReminderRepository
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler(config *Config, logger *zap.Logger, db *gorm.DB) (*Scheduler, error) {
	// Connect to Redis
	redisOpts := &redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create repositories
	taskRepo := database.TaskRepository(db)
	reminderRepo := database.ReminderRepository(db)

	return &Scheduler{
		config:    config,
		logger:    logger,
		db:        db,
		redis:     redisClient,
		tasks:     taskRepo,
		reminders: reminderRepo,
		stopCh:    make(chan struct{}),
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.logger.Info("Starting scheduler")

	// Start the polling goroutine
	s.wg.Add(1)
	go s.pollTasks()

	// Start the reminder processing goroutine
	s.wg.Add(1)
	go s.processReminders()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping scheduler")

	// Signal all goroutines to stop
	close(s.stopCh)

	// Wait for all goroutines to finish
	s.wg.Wait()

	// Close Redis connection
	if err := s.redis.Close(); err != nil {
		s.logger.Error("Failed to close Redis connection", zap.Error(err))
	}

	s.logger.Info("Scheduler stopped")
	return nil
}

// pollTasks polls for tasks that need to be scheduled
func (s *Scheduler) pollTasks() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.checkDueTasks(); err != nil {
				s.logger.Error("Failed to check due tasks", zap.Error(err))
			}
		case <-s.stopCh:
			return
		}
	}
}

// checkDueTasks checks for tasks that are due and creates reminders for them
func (s *Scheduler) checkDueTasks() error {
	s.logger.Debug("Checking for due tasks")

	// Get tasks that are due
	ctx := context.Background()
	tasks, err := s.tasks.ListDue(ctx, 0, s.config.MaxConcurrentTasks)
	if err != nil {
		return fmt.Errorf("failed to list due tasks: %w", err)
	}

	// Process each task
	for _, task := range tasks {
		if err := s.createReminder(ctx, task); err != nil {
			s.logger.Error("Failed to create reminder for task",
				zap.Uint("task_id", task.ID),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// createReminder creates a reminder for a task
func (s *Scheduler) createReminder(ctx context.Context, task *models.TaskInstance) error {
	s.logger.Info("Creating reminder for task", zap.Uint("task_id", task.ID))

	// Create a reminder
	reminder := &models.Reminder{
		TaskID: task.ID,
		ChatAt: time.Now(),
		State:  models.ReminderStatePending,
	}

	// Save the reminder
	if err := s.reminders.Create(ctx, reminder); err != nil {
		return fmt.Errorf("failed to create reminder: %w", err)
	}

	// Update the task state
	task.State = models.TaskStateScheduled
	if err := s.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	return nil
}

// processReminders processes reminders that are due
func (s *Scheduler) processReminders() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.checkDueReminders(); err != nil {
				s.logger.Error("Failed to check due reminders", zap.Error(err))
			}
		case <-s.stopCh:
			return
		}
	}
}

// checkDueReminders checks for reminders that are due and processes them
func (s *Scheduler) checkDueReminders() error {
	s.logger.Debug("Checking for due reminders")

	// Get reminders that are due
	ctx := context.Background()
	reminders, err := s.reminders.ListPending(ctx, 0, s.config.MaxConcurrentTasks)
	if err != nil {
		return fmt.Errorf("failed to list pending reminders: %w", err)
	}

	// Process each reminder
	for _, reminder := range reminders {
		if reminder.ChatAt.After(time.Now()) {
			continue
		}

		if err := s.sendReminder(ctx, reminder); err != nil {
			s.logger.Error("Failed to send reminder",
				zap.Uint("reminder_id", reminder.ID),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// sendReminder sends a reminder to the appropriate chat platform
func (s *Scheduler) sendReminder(ctx context.Context, reminder *models.Reminder) error {
	s.logger.Info("Sending reminder", zap.Uint("reminder_id", reminder.ID))

	// TODO: Implement sending reminders to chat platforms
	// This would involve:
	// 1. Getting the task details
	// 2. Determining the chat platform (Slack, Google Chat)
	// 3. Sending the message with interactive buttons
	// 4. Updating the reminder state

	// For now, just update the reminder state
	reminder.State = models.ReminderStateDelivered
	if err := s.reminders.Update(ctx, reminder); err != nil {
		return fmt.Errorf("failed to update reminder state: %w", err)
	}

	return nil
}

// ScheduleTask schedules a task for execution at a specific time
func (s *Scheduler) ScheduleTask(ctx context.Context, taskID uint, dueAt time.Time) error {
	s.logger.Info("Scheduling task", zap.Uint("task_id", taskID), zap.Time("due_at", dueAt))

	// Get the task
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update the task
	task.DueAt = &dueAt
	task.State = models.TaskStatePending
	if err := s.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// CancelTask cancels a scheduled task
func (s *Scheduler) CancelTask(ctx context.Context, taskID uint) error {
	s.logger.Info("Cancelling task", zap.Uint("task_id", taskID))

	// Get the task
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update the task
	task.State = models.TaskStateCancelled
	if err := s.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Cancel any pending reminders
	reminders, err := s.reminders.ListByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to list reminders: %w", err)
	}

	for _, reminder := range reminders {
		if reminder.State == models.ReminderStatePending {
			reminder.State = models.ReminderStateCancelled
			reminder.CancelledAt = timePtr(time.Now())
			if err := s.reminders.Update(ctx, reminder); err != nil {
				s.logger.Error("Failed to cancel reminder",
					zap.Uint("reminder_id", reminder.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
