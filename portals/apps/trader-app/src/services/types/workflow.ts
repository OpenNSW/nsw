import type { WorkflowNodeState } from './consignment'

export type WorkflowStepType = 'SIMPLE_FORM' | 'WAIT_FOR_EVENT' | 'PAYMENT'

export interface WorkflowStepConfig {
  formId?: string
  agency?: string
  service?: string
  event?: string
}

export interface WorkflowStep {
  stepId: string
  type: WorkflowStepType
  config: WorkflowStepConfig
  dependsOn: string[]
}

export interface WorkflowTemplate {
  id: string
  createdAt: string
  updatedAt: string
  version: string
  steps: WorkflowStep[]
}

export interface Workflow {
  id: string
  name: string
  type: 'import' | 'export'
  steps: WorkflowStep[]
}

export interface WorkflowQueryParams {
  hs_code: string
}

// Workflow V2 Types
export type WorkflowNodeType = 'TASK' | 'INTERNAL'
export type InternalNodeType = 'EVENT' | 'GATEWAY'
export type EventNodeType = 'START' | 'END'
export type GatewayNodeType = 'EXCLUSIVE_SPLIT' | 'EXCLUSIVE_JOIN' | 'PARALLEL_SPLIT' | 'PARALLEL_JOIN'

export interface WorkflowNodeV2 {
  id: string
  type: WorkflowNodeType
  name: string
  x: number
  y: number
  // For TASK
  task_id?: string
  output_mapping?: Record<string, string>
  // For INTERNAL
  internal_type?: InternalNodeType
  event_type?: EventNodeType
  gateway_type?: GatewayNodeType
  // State from execution
  state?: WorkflowNodeState
  extendedState?: string
}

export interface WorkflowEdgeV2 {
  id: string
  source_id: string
  target_id: string
  condition?: string
}

export interface WorkflowV2 {
  workflow_id: string
  name: string
  version: number
  nodes: WorkflowNodeV2[]
  edges: WorkflowEdgeV2[]
}