package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConsignmentService struct {
	ts *TaskService
	db *gorm.DB
}

func NewConsignmentService(ts *TaskService, db *gorm.DB) *ConsignmentService {
	return &ConsignmentService{ts: ts, db: db}
}

// InitializeConsignment creates a new consignment with all associated tasks based on workflow templates
func (s *ConsignmentService) InitializeConsignment(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.Consignment, error) {
	if createReq == nil {
		return nil, fmt.Errorf("create request cannot be nil")
	}
	if len(createReq.Items) == 0 {
		return nil, fmt.Errorf("consignment must have at least one item")
	}
	if createReq.TraderID == "" {
		return nil, fmt.Errorf("trader ID cannot be empty")
	}

	// Use a transaction to ensure atomicity
	return s.initializeConsignmentInTx(ctx, createReq)
}

func (s *ConsignmentService) initializeConsignmentInTx(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.Consignment, error) {
	var consignment *model.Consignment

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Convert CreateWorkflowForItemDTO to Item
		items := make([]model.Item, len(createReq.Items))
		for i, itemDTO := range createReq.Items {
			workflowTemplateID, err := uuid.Parse(itemDTO.WorkflowTemplateID)
			if err != nil {
				return fmt.Errorf("invalid workflow template ID for item %d: %w", i, err)
			}
			items[i] = model.Item{
				HSCode:             itemDTO.HSCode,
				WorkflowTemplateID: workflowTemplateID,
				Tasks:              []uuid.UUID{},
			}
		}

		consignment = &model.Consignment{
			Type:     createReq.Type,
			Items:    items,
			TraderID: createReq.TraderID,
			State:    model.ConsignmentStateInProgress,
		}

		// Save the consignment to generate an ID
		if err := tx.Create(consignment).Error; err != nil {
			return fmt.Errorf("failed to create consignment: %w", err)
		}

		// Process each item in the consignment
		for itemIdx := range consignment.Items {
			item := &consignment.Items[itemIdx]

			// Query the workflow template for this item
			var workflowTemplate model.WorkflowTemplate
			if err := tx.First(&workflowTemplate, "id = ?", item.WorkflowTemplateID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("workflow template %s not found for item with HS code %s", item.WorkflowTemplateID, item.HSCode)
				}
				return fmt.Errorf("failed to query workflow template: %w", err)
			}

			// Build tasks for this item
			tasks, err := s.buildTasksFromTemplate(consignment.ID, workflowTemplate)
			if err != nil {
				return fmt.Errorf("failed to build tasks for item %d: %w", itemIdx, err)
			}

			// Save all tasks for this item using the transaction
			taskIDs, err := s.ts.CreateTasksInTx(ctx, tx, tasks)
			if err != nil {
				return fmt.Errorf("failed to create tasks for item %d: %w", itemIdx, err)
			}

			// Store task IDs in the item
			item.Tasks = taskIDs
		}

		// Update the consignment with the task IDs
		if err := tx.Save(consignment).Error; err != nil {
			return fmt.Errorf("failed to update consignment with task IDs: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return consignment, nil
}

// buildTasksFromTemplate creates task instances from a workflow template
func (s *ConsignmentService) buildTasksFromTemplate(consignmentID uuid.UUID, template model.WorkflowTemplate) ([]model.Task, error) {
	if len(template.Steps) == 0 {
		return nil, fmt.Errorf("workflow template has no steps")
	}

	tasks := make([]model.Task, 0, len(template.Steps))

	for _, step := range template.Steps {
		// Determine task status based on dependencies
		status := model.TaskStatusReady
		dependsOnMap := make(map[string]model.DependencyStatus)

		if len(step.DependsOn) > 0 {
			status = model.TaskStatusLocked
			// Initialize all dependencies as INCOMPLETE
			for _, depStepID := range step.DependsOn {
				dependsOnMap[depStepID] = model.DependencyStatusIncomplete
			}
		}

		// Create the task
		task := model.Task{
			ConsignmentID: consignmentID,
			StepID:        step.StepID,
			Type:          step.Type,
			Status:        status,
			Config:        step.Config,
			DependsOn:     dependsOnMap,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateTaskStatusAndPropagateChanges updates a task's status and propagates changes to dependent tasks and consignment state and return updated dependent tasks that state became READY
func (s *ConsignmentService) UpdateTaskStatusAndPropagateChanges(ctx context.Context, taskID uuid.UUID, newStatus model.TaskStatus) ([]model.InitTaskInTaskManagerDTO, error) {
	if taskID == uuid.Nil {
		return nil, fmt.Errorf("task ID cannot be nil")
	}
	if newStatus == "" {
		return nil, fmt.Errorf("task status cannot be empty")
	}

	var newReadyStateDependentTasks []model.InitTaskInTaskManagerDTO

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Retrieve the task to be updated
		task, err := s.ts.GetTaskByIDInTx(ctx, tx, taskID)
		if err != nil {
			return err
		}

		// Update the task status
		task.Status = newStatus
		if err := s.ts.UpdateTaskInTx(ctx, tx, task); err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		// Update dependent tasks
		readyDependentTasks, err := s.updateDependentTasks(ctx, tx, *task)
		if err != nil {
			return fmt.Errorf("failed to update dependent tasks: %w", err)
		}
		newReadyStateDependentTasks = readyDependentTasks

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newReadyStateDependentTasks, nil
}

// updateDependentTasks marks the completed task as COMPLETED in all dependent tasks' DependsOn maps
func (s *ConsignmentService) updateDependentTasks(ctx context.Context, tx *gorm.DB, completedTask model.Task) ([]model.InitTaskInTaskManagerDTO, error) {
	// If the completed task is not marked as COMPLETED, no need to update dependents
	if completedTask.Status != model.TaskStatusCompleted {
		return []model.InitTaskInTaskManagerDTO{}, nil
	}

	// If the completed task has been marked as REJECTED, need to update consignment state to REQUIRES_REWORK
	if completedTask.Status == model.TaskStatusRejected {
		consignment, err := s.GetConsignmentByID(ctx, completedTask.ConsignmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve consignment: %w", err)
		}
		consignment.State = model.ConsignmentStateRequiresRework
		if err := tx.Save(&consignment).Error; err != nil {
			return nil, fmt.Errorf("failed to update consignment state: %w", err)
		}
		return []model.InitTaskInTaskManagerDTO{}, nil
	}

	// Get all tasks in the same consignment
	allTasks, err := s.ts.GetTasksByConsignmentID(ctx, completedTask.ConsignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tasks for consignment %s: %w", completedTask.ConsignmentID, err)
	}

	// Collect tasks that need updates for batch processing
	tasksToUpdate := make([]*model.Task, 0)

	// Collect the dependent tasks that became READY
	var readyDependentTasks []model.InitTaskInTaskManagerDTO

	// Variable to track if consignment state needs to be updated
	isAllCompleted := true

	// Find tasks that depend on the completed task
	for i := range allTasks {
		dependentTask := &allTasks[i]

		// Check if this task depends on the completed task
		if _, exists := dependentTask.DependsOn[completedTask.StepID]; exists {
			// Mark this dependency as completed
			dependentTask.DependsOn[completedTask.StepID] = model.DependencyStatusCompleted

			// Check if all dependencies are now completed
			allDepsCompleted := true
			for _, status := range dependentTask.DependsOn {
				if status == model.DependencyStatusIncomplete {
					allDepsCompleted = false
					break
				}
			}

			// If all dependencies are completed and task was locked, make it ready
			if allDepsCompleted && dependentTask.Status == model.TaskStatusLocked {
				dependentTask.Status = model.TaskStatusReady
				readyDependentTasks = append(readyDependentTasks, model.InitTaskInTaskManagerDTO{
					TaskID: dependentTask.ID,
					Type:   dependentTask.Type,
					Status: dependentTask.Status,
					Config: dependentTask.Config,
				})
			}

			tasksToUpdate = append(tasksToUpdate, dependentTask)
		}
		if dependentTask.Status != model.TaskStatusCompleted {
			isAllCompleted = false
		}
	}

	// Batch update all modified tasks using TaskService
	if len(tasksToUpdate) > 0 {
		if err := s.ts.UpdateTasksInTx(ctx, tx, tasksToUpdate); err != nil {
			return nil, fmt.Errorf("failed to update dependent tasks: %w", err)
		}
	}

	// Update consignment state based on task completions
	if isAllCompleted {
		consignment, err := s.GetConsignmentByID(ctx, completedTask.ConsignmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve consignment: %w", err)
		}
		consignment.State = model.ConsignmentStateFinished
		if err := tx.Save(&consignment).Error; err != nil {
			return nil, fmt.Errorf("failed to update consignment state: %w", err)
		}
	}
	return readyDependentTasks, nil
}

// GetConsignmentByID retrieves a consignment by its ID.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.Consignment, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}

	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("consignment %s not found", consignmentID)
		}
		return nil, fmt.Errorf("failed to retrieve consignment: %w", result.Error)
	}
	return &consignment, nil
}
