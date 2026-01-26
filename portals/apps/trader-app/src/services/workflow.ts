import { apiGet } from './api'
import type { Workflow, WorkflowTemplate, WorkflowQueryParams } from './types/workflow'

export interface WorkflowResponse {
  import: Workflow[]
  export: Workflow[]
}

const WORKFLOW_API_URL = 'http://localhost:8080/api/workflows/templates'

export async function getWorkflowsByHSCode(
  params: WorkflowQueryParams
): Promise<WorkflowResponse> {

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

  return apiGet<Workflow>(`/workflows/${id}`)
}