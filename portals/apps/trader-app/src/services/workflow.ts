import { apiGet, USE_MOCK } from './api'
import type { Workflow, WorkflowTemplate, WorkflowQueryParams } from './types/workflow'
import { mockWorkflows } from './mocks/workflowData'

export interface WorkflowResponse {
  import: Workflow[]
  export: Workflow[]
}

function getMockWorkflows(params: WorkflowQueryParams): WorkflowResponse {
  const { hs_code } = params

  // Find workflows where the workflow's hsCode starts with the searched code
  // This allows searching for parent codes to find child workflows
  // e.g., searching "0902" returns workflows for "090210", "090220", etc.
  const workflows = mockWorkflows.filter((wf) => wf.hsCode.startsWith(hs_code))

  return {
    import: workflows.filter((wf) => wf.type === 'import'),
    export: workflows.filter((wf) => wf.type === 'export'),
  }
}

const WORKFLOW_API_URL = 'http://localhost:8080/api/workflow-template'

export async function getWorkflowsByHSCode(
  params: WorkflowQueryParams
): Promise<WorkflowResponse> {
  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 200))
    return getMockWorkflows(params)
  }

  // Fetch import and export workflows in parallel
  const [importWorkflow, exportWorkflow] = await Promise.all([
    fetchWorkflowByType(params.hs_code, 'IMPORT'),
    fetchWorkflowByType(params.hs_code, 'EXPORT'),
  ])

  return {
    import: importWorkflow ? [importWorkflow] : [],
    export: exportWorkflow ? [exportWorkflow] : [],
  }
}

async function fetchWorkflowByType(
  hsCode: string,
  tradeFlow: 'IMPORT' | 'EXPORT'
): Promise<Workflow | null> {
  const url = `${WORKFLOW_API_URL}?hsCode=${encodeURIComponent(hsCode)}&tradeFlow=${tradeFlow}`
  const response = await fetch(url)

  if (!response.ok) {
    if (response.status === 404) {
      return null
    }
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  const template: WorkflowTemplate = await response.json()

  // Transform WorkflowTemplate to Workflow
  return {
    id: template.id,
    name: template.version,
    type: tradeFlow.toLowerCase() as 'import' | 'export',
    steps: template.steps,
  }
}

export async function getWorkflowById(id: string): Promise<Workflow | undefined> {
  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 200))
    return mockWorkflows.find((wf) => wf.id === id)
  }

  return apiGet<Workflow>(`/workflows/${id}`)
}