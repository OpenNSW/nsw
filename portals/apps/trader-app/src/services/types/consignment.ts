export type TradeFlow = 'IMPORT' | 'EXPORT'

export type ConsignmentState = 'AWAITING_INITIATION' | 'IN_PROGRESS' | 'FINISHED' | 'COMPLETED' | 'REQUIRES_REWORK'

export type WorkflowNodeState = 'READY' | 'LOCKED' | 'IN_PROGRESS' | 'COMPLETED' | 'REJECTED'

export type StepType = 'SIMPLE_FORM' | 'WAIT_FOR_EVENT'

export interface GlobalContext {
  consigneeAddress: string
  consigneeName: string
  countryOfDestination: string
  countryOfOrigin: string
  invoiceDate: string
  invoiceNumber: string
}

export interface CustomsHouseAgent {
  id: string
  name: string
  description: string
}

export interface HSCodeDetails {
  hsCodeId: string
  hsCode: string
  description: string
  category: string
}

export interface WorkflowNodeTemplate {
  name: string
  description: string
  type: StepType
}

export interface WorkflowNode {
  id: string
  createdAt: string
  updatedAt: string
  workflowNodeTemplate: WorkflowNodeTemplate
  state: WorkflowNodeState
  extendedState?: string
  depends_on: string[]
}

export interface ConsignmentItem {
  hsCode: HSCodeDetails
}


export interface ConsignmentSummary {
  id: string
  flow: TradeFlow
  traderId: string
  state: ConsignmentState
  items: ConsignmentItem[]
  createdAt: string
  updatedAt: string
  chaId?: string
  workflowNodeCount: number
  completedWorkflowNodeCount: number
}

export interface ConsignmentDetail {
  id: string
  flow: TradeFlow
  traderId: string
  state: ConsignmentState
  items: ConsignmentItem[]
  globalContext: GlobalContext
  createdAt: string
  updatedAt: string
  chaId?: string
  cha?: CustomsHouseAgent
  workflowNodes: WorkflowNode[]
  /** Set after CHA initializes with HS Code (stage 2) */
  hsCodeId?: string
}

// Deprecated: Use ConsignmentDetail or ConsignmentSummary
export type Consignment = ConsignmentDetail

export interface CreateConsignmentItemRequest {
  hsCodeId: string
}

/** Stage 1 (Trader): create consignment shell with selected CHA */
export interface CreateConsignmentRequest {
  flow: TradeFlow
  chaId: string
}

/** Stage 2 (CHA): initialize consignment with HS Code(s) */
export interface InitializeConsignmentRequest {
  items: CreateConsignmentItemRequest[]
}

export type CreateConsignmentResponse = ConsignmentDetail

import type { PaginatedResponse } from './common'

export type ConsignmentListResult = PaginatedResponse<ConsignmentSummary>