import { apiGet, apiPost, USE_MOCK } from './api'
import { findTaskDetails, type TaskDetails } from './mocks/taskData'
import type { StepType } from './types/consignment'
import type {JsonSchema, UISchemaElement} from "../components/JsonForm";

export type TaskAction = 'FETCH_FORM' | 'SUBMIT_FORM' | 'DRAFT'

export interface ExecuteTaskRequest {
  task_id: string
  consignment_id: string
  payload: {
    action: TaskAction
    data?: Record<string, unknown>
  }
}

export interface TaskFormData {
  title: string
  schema: JsonSchema
  uiSchema: UISchemaElement
  formData: Record<string, unknown>
}

export interface ExecuteTaskResult {
  status: string
  message: string
  data: TaskFormData
}

export interface ExecuteTaskResponse {
  success: boolean
  result: ExecuteTaskResult
}

export type TaskCommand = 'SUBMISSION' | 'DRAFT'

export interface TaskCommandRequest {
  command: TaskCommand
  taskId: string
  consignmentId: string
  data: Record<string, unknown>
}

export interface TaskCommandResponse {
  success: boolean
  message: string
  taskId: string
  status?: string
}

const TASKS_API_URL = 'http://localhost:8080/api/tasks'

function getActionForStepType(stepType: StepType): TaskAction {
  switch (stepType) {
    case 'TRADER_FORM':
    case 'OGA_FORM':
      return 'FETCH_FORM'
    default:
      return 'FETCH_FORM'
  }
}

export async function executeTask(
  consignmentId: string,
  taskId: string,
  stepType: StepType
): Promise<ExecuteTaskResponse> {
  const action = getActionForStepType(stepType)

  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 500))

    // Mock response
    return {
      success: true,
      result: {
        status: 'READY',
        message: 'Form schema retrieved successfully',
        data: {
          title: 'CUSDEC I & II Declaration Form',
          schema: {
            type: 'object',
            properties: {
              header: {
                type: 'object',
                title: 'General Information'
              },
            },
          },
          uiSchema: {
            type: 'VerticalLayout',
            elements: [
              {
                type: 'Group',
                label: 'General Information',
                elements: [
                  { type: 'Control', scope: '#/properties/header/properties/declarationType' },
                  { type: 'Control', scope: '#/properties/header/properties/officeOfEntry' },
                  { type: 'Control', scope: '#/properties/header/properties/customsReference' },
                  { type: 'Control', scope: '#/properties/header/properties/date' },
                ],
              },
            ],
          },
          formData: {},
        },
      },
    }
  }

  const response = await fetch(TASKS_API_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      task_id: taskId,
      consignment_id: consignmentId,
      payload: {
        action,
      },
    }),
  })

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return response.json()
}

export async function getTaskDetails(
  consignmentId: string,
  taskId: string
): Promise<TaskDetails> {
  console.log(
    `Fetching task details for consignment: ${consignmentId}, task: ${taskId}`
  )

  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 300))

    // Find the task details based on taskId
    const taskDetails = findTaskDetails(taskId)
    if (!taskDetails) {
      throw new Error(`Task not found: ${taskId}`)
    }

    return taskDetails
  }

  return apiGet<TaskDetails>(`/workflows/${consignmentId}/tasks/${taskId}`)
}

export async function sendTaskCommand(
  request: TaskCommandRequest
): Promise<TaskCommandResponse> {
  console.log(`Sending ${request.command} command for task: ${request.taskId}`, request)

  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 500))

    // Mock successful response
    return {
      success: true,
      message: request.command === 'DRAFT'
        ? 'Draft saved successfully'
        : 'Task submitted successfully',
      taskId: request.taskId,
      status: request.command === 'DRAFT' ? 'DRAFT' : 'SUBMITTED',
    }
  }

  return apiPost<TaskCommandRequest, TaskCommandResponse>(
    `/tasks/${request.taskId}/command`,
    request
  )
}