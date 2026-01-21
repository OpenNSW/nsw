import { apiGet, USE_MOCK } from './api'
import type { TaskDetails } from './mocks/taskData'
import { mockTaskDetails } from './mocks/taskData'

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
    // In a real app, you'd use the consignmentId and taskId to fetch
    // the specific task. For this mock, we return the same details.
    return mockTaskDetails
  }

  return apiGet<TaskDetails>(`/workflows/${consignmentId}/tasks/${taskId}`)
}
