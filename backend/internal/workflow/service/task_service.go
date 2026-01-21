package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskService struct {
	db *gorm.DB
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

// CreateTasks creates multiple tasks in the database.
func (s *TaskService) CreateTasks(ctx context.Context, tasks []model.Task) ([]uuid.UUID, error) {
	if len(tasks) == 0 {
		return []uuid.UUID{}, nil
	}

	result := s.db.WithContext(ctx).Create(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create tasks: %w", result.Error)
	}

	taskIDs := make([]uuid.UUID, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}

	return taskIDs, nil
}

// GetTasksByConsignmentID retrieves all tasks associated with a given consignment ID.
func (s *TaskService) GetTasksByConsignmentID(ctx context.Context, consignmentID uuid.UUID) ([]model.Task, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}

	var tasks []model.Task
	result := s.db.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve tasks: %w", result.Error)
	}
	// Return empty slice instead of error when no tasks found
	return tasks, nil
}

// GetTaskByID retrieves a task by its ID.
func (s *TaskService) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
	if taskID == uuid.Nil {
		return nil, fmt.Errorf("task ID cannot be nil")
	}

	var task model.Task
	result := s.db.WithContext(ctx).First(&task, "id = ?", taskID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to retrieve task: %w", result.Error)
	}
	return &task, nil
}

// UpdateTasks updates multiple tasks in the database within a transaction.
func (s *TaskService) UpdateTasks(ctx context.Context, tasks []model.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range tasks {
			if err := tx.Save(&tasks[i]).Error; err != nil {
				return fmt.Errorf("failed to update task %s: %w", tasks[i].ID, err)
			}
		}
		return nil
	})
}

// updateTasksInTx updates multiple tasks within an existing transaction.
// This is used internally when we're already in a transaction context.
func (s *TaskService) updateTasksInTx(ctx context.Context, tx *gorm.DB, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if err := tx.WithContext(ctx).Save(task).Error; err != nil {
			return fmt.Errorf("failed to update task %s: %w", task.ID, err)
		}
	}
	return nil
}

// UpdateTaskStatus updates a single task's status.
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error {
	if taskID == uuid.Nil {
		return fmt.Errorf("task ID cannot be nil")
	}
	if status == "" {
		return fmt.Errorf("task status cannot be empty")
	}

	result := s.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ?", taskID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update task status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task %s not found", taskID)
	}

	return nil
}
