package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/utils"
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

// GetAllHSCodes retrieves all HS codes from the database
func (s *ConsignmentService) GetAllHSCodes(ctx context.Context, filter model.HSCodeFilter) (*model.HSCodeListResult, error) {
	// Get total count first for pagination (with filter applied)
	var totalCount int64
	countQuery := s.db.WithContext(ctx).Model(&model.HSCode{})

	// Apply the same filter to the count query
	if filter.HSCodeStartsWith != nil && *filter.HSCodeStartsWith != "" {
		countQuery = countQuery.Where("hs_code LIKE ?", *filter.HSCodeStartsWith+"%")
	}

	countResult := countQuery.Count(&totalCount)
	if countResult.Error != nil {
		return nil, fmt.Errorf("failed to count HS codes: %w", countResult.Error)
	}

	// If no HS codes found, return early
	if totalCount == 0 {
		return &model.HSCodeListResult{
			TotalCount: 0,
			Items:      []model.HSCode{},
			Offset:     0,
			Limit:      0,
		}, nil
	}

	var hsCodes []model.HSCode
	query := s.db.WithContext(ctx)

	// Apply filter: HSCode starts with
	if filter.HSCodeStartsWith != nil && *filter.HSCodeStartsWith != "" {
		query = query.Where("hs_code LIKE ?", *filter.HSCodeStartsWith+"%")
	}

	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(filter.Offset, filter.Limit)
	query = query.Offset(finalOffset).Limit(finalLimit)

	// Add ordering for consistent pagination
	query = query.Order("hs_code ASC")

	result := query.Find(&hsCodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve HS codes: %w", result.Error)
	}

	// Prepare the result
	hsCodeListResult := &model.HSCodeListResult{
		TotalCount: totalCount,
		Items:      hsCodes,
		Offset:     finalOffset,
		Limit:      finalLimit,
	}

	return hsCodeListResult, nil
}

// GetHSCodeByID retrieves an HS code by its ID from the database
func (s *ConsignmentService) GetHSCodeByID(ctx context.Context, hsCodeID uuid.UUID) (*model.HSCode, error) {
	var hsCode model.HSCode
	result := s.db.WithContext(ctx).First(&hsCode, "id = ?", hsCodeID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("HS code with ID %s not found", hsCodeID)
		}
		return nil, fmt.Errorf("failed to retrieve HS code: %w", result.Error)
	}
	return &hsCode, nil
}

// GetWorkFlowTemplate retrieves a workflow template based on HS code and consignment type
func (s *ConsignmentService) GetWorkFlowTemplate(ctx context.Context, hsCode *string, hsCodeID *uuid.UUID, tradeFlow model.TradeFlow) (*model.WorkflowTemplate, error) {
	if hsCode == nil && hsCodeID == nil {
		return nil, fmt.Errorf("either hscode or hscodeID must be provided")
	}

	query := s.db.WithContext(ctx)

	if hsCodeID != nil {
		// Query by HS code ID
		var workflowTemplate model.WorkflowTemplate
		err := query.
			Joins("JOIN workflow_template_maps ON workflow_templates.id = workflow_template_maps.workflow_template_id").
			Where("workflow_template_maps.hs_code_id = ? AND workflow_template_maps.trade_flow = ?", *hsCodeID, tradeFlow).
			First(&workflowTemplate).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("workflow template not found for HS code ID %s and trade flow %s", *hsCodeID, tradeFlow)
			}
			return nil, fmt.Errorf("failed to retrieve workflow template: %w", err)
		}

		return &workflowTemplate, nil
	}

	// SELECT workflow_templates.* FROM workflow_templates
	// JOIN workflow_template_maps ON workflow_templates.id = workflow_template_maps.workflow_template_id
	// JOIN hs_codes ON hs_codes.id = workflow_template_maps.hs_code_id
	// WHERE hs_codes.code = ? AND workflow_template_maps.trade_flow = ?
	var workflowTemplate model.WorkflowTemplate
	err := query.
		Joins("JOIN workflow_template_maps ON workflow_templates.id = workflow_template_maps.workflow_template_id").
		Joins("JOIN hs_codes ON hs_codes.id = workflow_template_maps.hs_code_id").
		Where("hs_codes.hs_code = ? AND workflow_template_maps.trade_flow = ?", *hsCode, tradeFlow).
		First(&workflowTemplate).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow template not found for HS code %s and trade flow %s", *hsCode, tradeFlow)
		}
		return nil, fmt.Errorf("failed to retrieve workflow template: %w", err)
	}

	return &workflowTemplate, nil
}

// InitializeConsignment creates a new consignment with all associated tasks based on workflow templates
// Returns the created consignment and a list of ready tasks that should be registered with Task Manager
func (s *ConsignmentService) InitializeConsignment(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.Consignment, []*model.Task, error) {
	if createReq == nil {
		return nil, nil, fmt.Errorf("create request cannot be nil")
	}
	if len(createReq.Items) == 0 {
		return nil, nil, fmt.Errorf("consignment must have at least one item")
	}
	if createReq.TraderID == nil {
		return nil, nil, fmt.Errorf("trader ID cannot be empty")
	}

	// Use a transaction to ensure atomicity
	return s.initializeConsignmentInTx(ctx, createReq)
}

func (s *ConsignmentService) initializeConsignmentInTx(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.Consignment, []*model.Task, error) {
	var consignment *model.Consignment
	var readyTasks []*model.Task

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Fetch workflow templates for all items (with transaction context and preload Steps)
		// Always query by HSCodeID (required field) and validate against WorkflowTemplateID if provided
		workflowTemplates := make([]*model.WorkflowTemplate, len(createReq.Items))
		for i := range createReq.Items {
			// Query workflow template by HS code ID and trade flow
			workflowTemplate, err := s.GetWorkFlowTemplate(ctx, nil, &createReq.Items[i].HSCodeID, createReq.TradeFlow)
			if err != nil {
				return fmt.Errorf("failed to get workflow template for item %d: %w", i, err)
			}
			workflowTemplates[i] = workflowTemplate
		}

		// Initialize items with HSCodeID only - Steps will be populated after tasks are created
		items := make([]model.Item, len(createReq.Items))
		for i := range createReq.Items {
			// fetch HSCode details from HSCodeID
			hsCode, err := s.GetHSCodeByID(ctx, createReq.Items[i].HSCodeID)
			if err != nil {
				return fmt.Errorf("failed to get HS code for item %d: %w", i, err)
			}

			items[i] = model.Item{
				HSCodeID:          createReq.Items[i].HSCodeID,
				HSCode:            hsCode.HSCode,
				HSCodeDescription: hsCode.Description,
				AdditionalData:    createReq.Items[i].ItemData,
				Steps:             []model.ConsignmentStep{},
			}
		}

		// Prepare global context
		globalContext := make(map[string]interface{})
		if createReq.GlobalContext != nil {
			globalContext = createReq.GlobalContext
		}

		consignment = &model.Consignment{
			TradeFlow:     createReq.TradeFlow,
			Items:         items,
			TraderID:      *createReq.TraderID,
			State:         model.ConsignmentStateInProgress,
			GlobalContext: globalContext,
		}

		// Save the consignment to generate an ID
		if err := tx.Create(consignment).Error; err != nil {
			return fmt.Errorf("failed to create consignment: %w", err)
		}

		// Process each item in the consignment
		for itemIdx := range consignment.Items {
			item := &consignment.Items[itemIdx]
			workflowTemplate := workflowTemplates[itemIdx]

			// Build tasks for this item
			tasks, err := s.buildTasksFromTemplate(consignment.ID, *workflowTemplate)
			if err != nil {
				return fmt.Errorf("failed to build tasks for item %d: %w", itemIdx, err)
			}

			// Save all tasks for this item using the transaction
			taskIDs, err := s.ts.CreateTasksInTx(ctx, tx, tasks)
			if err != nil {
				return fmt.Errorf("failed to create tasks for item %d: %w", itemIdx, err)
			}

			// Build ConsignmentSteps from created tasks
			steps := make([]model.ConsignmentStep, len(tasks))
			for i, taskID := range taskIDs {
				tasks[i].ID = taskID // Set the generated ID

				// Collect ready tasks for registration with Task Manager
				if tasks[i].Status == model.TaskStatusReady {
					readyTasks = append(readyTasks, &tasks[i])
				}

				// Convert DependsOn map keys to slice for ConsignmentStep
				dependsOnSlice := make([]string, 0, len(tasks[i].DependsOn))
				for stepID := range tasks[i].DependsOn {
					dependsOnSlice = append(dependsOnSlice, stepID)
				}

				// Create ConsignmentStep
				steps[i] = model.ConsignmentStep{
					StepID:    tasks[i].StepID,
					Type:      tasks[i].Type,
					TaskID:    taskID,
					Status:    tasks[i].Status,
					DependsOn: dependsOnSlice,
				}
			}

			// Store steps in the item
			item.Steps = steps
		}

		// Update the consignment with the populated steps
		if err := tx.Save(consignment).Error; err != nil {
			return fmt.Errorf("failed to update consignment with steps: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return consignment, readyTasks, nil
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
func (s *ConsignmentService) UpdateTaskStatusAndPropagateChanges(ctx context.Context, taskID uuid.UUID, newStatus model.TaskStatus, consignmentGlobalContext map[string]interface{}) ([]*model.Task, *model.Consignment, error) {
	if taskID == uuid.Nil {
		return nil, nil, fmt.Errorf("task ID cannot be nil")
	}
	if newStatus == "" {
		return nil, nil, fmt.Errorf("task status cannot be empty")
	}

	var newReadyStateDependentTasks []*model.Task
	var consignment *model.Consignment

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Retrieve the task to be updated
		task, err := s.ts.GetTaskByIDInTx(ctx, tx, taskID)
		if err != nil {
			return err
		}

		if task.Status == newStatus && consignmentGlobalContext == nil {
			// No change in status and no global context update, nothing to do
			return nil
		}

		// If no change in task status but global context needs to be updated
		if task.Status == newStatus && consignmentGlobalContext != nil {
			// Only update the consignment with the global context
			consignment, err = s.updateConsignment(ctx, tx, task.ConsignmentID, nil, nil, &consignmentGlobalContext)
			if err != nil {
				return fmt.Errorf("failed to update consignment global context: %w", err)
			}
			return nil
		}

		// Update the task status
		task.Status = newStatus
		if err := s.ts.UpdateTaskInTx(ctx, tx, task); err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		// Update dependent tasks
		readyDependentTasks, stepsToUpdate, consignmentState, err := s.updateDependentTasks(ctx, tx, *task)
		if err != nil {
			return fmt.Errorf("failed to update dependent tasks: %w", err)
		}

		// Add current task's status update to the steps to update
		stepsToUpdate = append(stepsToUpdate, &model.StepStatusUpdateDTO{
			StepID:    task.StepID,
			NewStatus: task.Status,
		})

		// Update the consignment state, steps, and global context
		consignment, err = s.updateConsignment(ctx, tx, task.ConsignmentID, consignmentState, stepsToUpdate, &consignmentGlobalContext)
		if err != nil {
			return fmt.Errorf("failed to update consignment: %w", err)
		}

		// Return the newly READY dependent tasks
		newReadyStateDependentTasks = readyDependentTasks
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return newReadyStateDependentTasks, consignment, nil
}

// updateDependentTasks updates tasks that depend on the completed task and returns newly READY tasks and whether consignment is completed or steps to update
func (s *ConsignmentService) updateDependentTasks(ctx context.Context, tx *gorm.DB, completedTask model.Task) ([]*model.Task, []*model.StepStatusUpdateDTO, *model.ConsignmentState, error) {

	// If the completed task is not marked as COMPLETED, no need to update dependents
	if completedTask.Status != model.TaskStatusCompleted {
		return []*model.Task{}, []*model.StepStatusUpdateDTO{}, nil, nil
	}

	// Get all tasks in the same consignment
	allTasks, err := s.ts.GetTasksByConsignmentID(ctx, completedTask.ConsignmentID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to retrieve tasks for consignment %s: %w", completedTask.ConsignmentID, err)
	}

	// Collect tasks that need updates for batch processing
	tasksToUpdate := make([]*model.Task, 0)

	// Collect the steps that need to be updated in the consignment
	stepsToUpdate := make([]*model.StepStatusUpdateDTO, 0)

	// Collect the dependent tasks that became READY
	var readyDependentTasks []*model.Task

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
				readyDependentTasks = append(readyDependentTasks, dependentTask)
			}

			tasksToUpdate = append(tasksToUpdate, dependentTask)
			stepsToUpdate = append(stepsToUpdate, &model.StepStatusUpdateDTO{
				StepID:    dependentTask.StepID,
				NewStatus: dependentTask.Status,
			})
		}
		if dependentTask.Status != model.TaskStatusCompleted && dependentTask.ID != completedTask.ID {
			isAllCompleted = false
		}
	}

	// Batch update all modified tasks using TaskService
	if len(tasksToUpdate) > 0 {
		if err := s.ts.UpdateTasksInTx(ctx, tx, tasksToUpdate); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update dependent tasks: %w", err)
		}
	}

	consignmentStatus := model.ConsignmentStateInProgress
	// If all tasks in the consignment are completed, update consignment state to FINISHED
	if isAllCompleted {
		consignmentStatus = model.ConsignmentStateCompleted
	}

	return readyDependentTasks, stepsToUpdate, &consignmentStatus, nil
}

// getConsignmentByID retrieves a consignment by its ID from the database
func (s *ConsignmentService) getConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.Consignment, error) {
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

// updateConsignment updates the consignment state, steps status, and global context in a transaction
func (s *ConsignmentService) updateConsignment(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID, newState *model.ConsignmentState, stepsToUpdate []*model.StepStatusUpdateDTO, appendGlobalContext *map[string]interface{}) (*model.Consignment, error) {
	var consignment model.Consignment
	if err := tx.WithContext(ctx).First(&consignment, "id = ?", consignmentID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve consignment %s: %w", consignmentID, err)
	}

	// Update consignment state if provided
	if newState != nil {
		consignment.State = *newState
	}
	// Append to global context if provided
	if appendGlobalContext != nil {
		for k, v := range *appendGlobalContext {
			// TODO: Need to change back to erroring out if key already exists
			if _, exists := consignment.GlobalContext[k]; !exists {
				consignment.GlobalContext[k] = v
			}
		}
	}

	// Update steps if any
	if len(stepsToUpdate) > 0 {
		stepMap := make(map[string]*model.ConsignmentStep)
		for i := range consignment.Items {
			item := &consignment.Items[i]
			for j := range item.Steps {
				step := &item.Steps[j]
				stepMap[step.StepID] = step
			}
		}

		for _, stepUpdate := range stepsToUpdate {
			if step, ok := stepMap[stepUpdate.StepID]; ok {
				step.Status = stepUpdate.NewStatus
			}
		}
	}

	// Save the updated consignment
	if err := tx.WithContext(ctx).Save(&consignment).Error; err != nil {
		return nil, fmt.Errorf("failed to update consignment %s: %w", consignmentID, err)
	}

	return &consignment, nil
}

// GetConsignmentByID retrieves a consignment by its ID.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.ConsignmentResponse, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}

	consignment, err := s.getConsignmentByID(ctx, consignmentID)
	if err != nil {
		return nil, err
	}

	response := consignment.ToConsignmentResponse()
	return &response, nil
}

// GetConsignmentsByTraderID retrieves all consignments for a specific trader.
func (s *ConsignmentService) GetConsignmentsByTraderID(ctx context.Context, filter model.ConsignmentTraderFilter) (model.ConsignmentListResponseDTO, error) {
	if filter.TraderID == "" {
		return model.ConsignmentListResponseDTO{}, fmt.Errorf("trader ID cannot be empty")
	}

	// Get total count first for pagination
	var totalCount int64
	countResult := s.db.WithContext(ctx).Model(&model.Consignment{}).Where("trader_id = ?", filter.TraderID).Count(&totalCount)
	if countResult.Error != nil {
		return model.ConsignmentListResponseDTO{}, fmt.Errorf("failed to count consignments: %w", countResult.Error)
	}

	// If no consignments found, return early
	if totalCount == 0 {
		return model.ConsignmentListResponseDTO{
			TotalCount: 0,
			Items:      []model.ConsignmentResponse{},
			Offset:     0,
			Limit:      0,
		}, nil
	}

	var consignments []model.Consignment
	query := s.db.WithContext(ctx).Where("trader_id = ?", filter.TraderID).Order("created_at DESC")

	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(filter.Offset, filter.Limit)
	query = query.Offset(finalOffset).Limit(finalLimit)

	result := query.Find(&consignments)
	if result.Error != nil {
		return model.ConsignmentListResponseDTO{}, fmt.Errorf("failed to retrieve consignments: %w", result.Error)
	}

	// Convert to response DTOs
	responseItems := make([]model.ConsignmentResponse, len(consignments))
	for i, consignment := range consignments {
		responseItems[i] = consignment.ToConsignmentResponse()
	}

	// Prepare the result
	consignmentListResult := model.ConsignmentListResponseDTO{
		TotalCount: totalCount,
		Items:      responseItems,
		Offset:     finalOffset,
		Limit:      finalLimit,
	}

	return consignmentListResult, nil
}
