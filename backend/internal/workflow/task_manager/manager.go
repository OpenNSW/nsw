package task_manager

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// TaskManager handles task execution and status management
type TaskManager interface {
	// ExecuteTask executes a task by ID
	ExecuteTask(ctx context.Context, taskID uuid.UUID) (*TaskResult, error)

	// GetTaskFormSchema returns the form schema for a task (if applicable)
	GetTaskFormSchema(ctx context.Context, taskID uuid.UUID) (*model.FormTemplate, error)

	// UpdateTaskStatus updates the status of a task
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error

	// GetTasksByStatus retrieves tasks filtered by status
	GetTasksByStatus(ctx context.Context, status model.TaskStatus, consignmentID *uuid.UUID) ([]*model.Task, error)

	// BulkCreateTasks creates multiple tasks at once
	BulkCreateTasks(ctx context.Context, tasks []*model.Task) error

	// BulkUpdateTasks updates multiple tasks at once
	BulkUpdateTasks(ctx context.Context, tasks []*model.Task) error

	// OnTaskCompleted is called when a task completes (for async workflows)
	OnTaskCompleted(ctx context.Context, taskID uuid.UUID, result *TaskResult) error
}

type taskManager struct {
	db      *gorm.DB
	factory TaskFactory
	// workflowEngine WorkflowEngine // Will be injected in later PR
}

// NewTaskManager creates a new TaskManager instance
func NewTaskManager(db *gorm.DB, factory TaskFactory) TaskManager {
	return &taskManager{
		db:      db,
		factory: factory,
	}
}

func (tm *taskManager) ExecuteTask(ctx context.Context, taskID uuid.UUID) (*TaskResult, error) {
	// 1. Load task from database with relationships
	var taskModel model.Task
	if err := tm.db.WithContext(ctx).Preload("TraderFormTemplate").Preload("OGAOfficerFormTemplate").First(&taskModel, "id = ?", taskID).Error; err != nil {
		return nil, err
	}

	// 2. Create task instance from factory
	task, err := tm.factory.CreateTask(TaskType(taskModel.Type), &taskModel)
	if err != nil {
		return nil, err
	}

	// 3. Build task context
	// Determine form template based on task type
	var formTemplate *model.FormTemplate
	taskType := TaskType(taskModel.Type)
	if taskType == TaskTypeTraderForm {
		formTemplate = &taskModel.TraderFormTemplate
	} else if taskType == TaskTypeOGAForm {
		formTemplate = &taskModel.OGAOfficerFormTemplate
	}

	taskCtx := &TaskContext{
		TaskID:        taskID,
		ConsignmentID: taskModel.ConsignmentID,
		AssigneeID:    taskModel.TraderID, // Default to TraderID, can be overridden based on task type
		FormTemplate:  formTemplate,
		FormData:      make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
	}

	// 4. Check if task can execute
	canExecute, err := task.CanExecute(ctx, taskCtx)
	if err != nil || !canExecute {
		return nil, err
	}

	// 5. Execute task
	result, err := task.Execute(ctx, taskCtx)
	if err != nil {
		return nil, err
	}

	// 6. Update task status in database
	taskModel.Status = result.Status
	if err := tm.db.WithContext(ctx).Save(&taskModel).Error; err != nil {
		return nil, err
	}

	// 7. Notify Workflow Engine (sync or async - will be implemented in later PR)
	// tm.notifyWorkflowEngine(ctx, taskID, result)

	return result, nil
}

func (tm *taskManager) GetTaskFormSchema(ctx context.Context, taskID uuid.UUID) (*model.FormTemplate, error) {
	var taskModel model.Task
	if err := tm.db.WithContext(ctx).Preload("TraderFormTemplate").Preload("OGAOfficerFormTemplate").First(&taskModel, "id = ?", taskID).Error; err != nil {
		return nil, err
	}

	// Return appropriate form template based on task type
	taskType := TaskType(taskModel.Type)
	if taskType == TaskTypeTraderForm {
		return &taskModel.TraderFormTemplate, nil
	} else if taskType == TaskTypeOGAForm {
		return &taskModel.OGAOfficerFormTemplate, nil
	}

	return nil, nil
}

func (tm *taskManager) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error {
	return tm.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", taskID).Update("status", status).Error
}

func (tm *taskManager) GetTasksByStatus(ctx context.Context, status model.TaskStatus, consignmentID *uuid.UUID) ([]*model.Task, error) {
	var tasks []*model.Task
	query := tm.db.WithContext(ctx).Where("status = ?", status)

	if consignmentID != nil {
		query = query.Where("consignment_id = ?", *consignmentID)
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

func (tm *taskManager) BulkCreateTasks(ctx context.Context, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	return tm.db.WithContext(ctx).Create(&tasks).Error
}

func (tm *taskManager) BulkUpdateTasks(ctx context.Context, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	for _, task := range tasks {
		if err := tm.db.WithContext(ctx).Save(task).Error; err != nil {
			return err
		}
	}
	return nil
}

func (tm *taskManager) OnTaskCompleted(ctx context.Context, taskID uuid.UUID, result *TaskResult) error {
	// Update task status
	if err := tm.UpdateTaskStatus(ctx, taskID, result.Status); err != nil {
		return err
	}

	// Notify Workflow Engine (async - will be implemented in later PR)
	// This is a placeholder for async notification logic

	return nil
}
