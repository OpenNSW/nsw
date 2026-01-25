import { USE_MOCK } from './api'
import type {
  Consignment,
  CreateConsignmentRequest,
  CreateConsignmentResponse,
  ConsignmentStep,
} from './types/consignment'

const CONSIGNMENT_API_URL = 'http://localhost:8080/api/consignments'

// In-memory store for mock consignments
const mockConsignments: Map<string, Consignment> = new Map()

// Sample mock steps
const mockSteps: ConsignmentStep[] = [
  {
    stepId: 'cusdec_entry',
    type: 'TRADER_FORM',
    taskId: 'task-001',
    status: 'COMPLETED',
    dependsOn: [],
  },
  {
    stepId: 'phytosanitary_cert',
    type: 'OGA_FORM',
    taskId: 'task-002',
    status: 'READY',
    dependsOn: ['cusdec_entry'],
  },
  {
    stepId: 'tea_blend_sheet',
    type: 'OGA_FORM',
    taskId: 'task-003',
    status: 'READY',
    dependsOn: ['cusdec_entry'],
  },
  {
    stepId: 'final_customs_clearance',
    type: 'WAIT_FOR_EVENT',
    taskId: 'task-004',
    status: 'LOCKED',
    dependsOn: ['phytosanitary_cert', 'tea_blend_sheet'],
  },
]

// Initialize with some sample data
const sampleConsignments: Consignment[] = [
  {
    id: 'CON-001',
    createdAt: '2024-01-15T10:30:00Z',
    updatedAt: '2024-01-18T14:20:00Z',
    tradeFlow: 'EXPORT',
    traderId: 'trader-123',
    state: 'COMPLETED',
    items: [{ hsCodeID: '09021011', steps: mockSteps }],
  },
  {
    id: 'CON-002',
    createdAt: '2024-01-16T09:15:00Z',
    updatedAt: '2024-01-17T11:45:00Z',
    tradeFlow: 'IMPORT',
    traderId: 'trader-123',
    state: 'IN_PROGRESS',
    items: [{ hsCodeID: '09023011', steps: mockSteps }],
  },
  {
    id: 'CON-003',
    createdAt: '2024-01-17T14:00:00Z',
    updatedAt: '2024-01-17T14:00:00Z',
    tradeFlow: 'EXPORT',
    traderId: 'trader-123',
    state: 'IN_PROGRESS',
    items: [{ hsCodeID: '09022019', steps: mockSteps }],
  },
]

// Initialize mock data
sampleConsignments.forEach((c) => mockConsignments.set(c.id, c))

function generateConsignmentId(): string {
  return crypto.randomUUID()
}

async function mockCreateConsignment(
  request: CreateConsignmentRequest
): Promise<CreateConsignmentResponse> {
  const consignmentId = generateConsignmentId()
  const now = new Date().toISOString()

  const consignment: Consignment = {
    id: consignmentId,
    createdAt: now,
    updatedAt: now,
    tradeFlow: request.tradeFlow,
    traderId: request.traderId,
    state: 'IN_PROGRESS',
    items: request.items.map((item) => ({
      hsCodeID: item.hsCodeId,
      steps: mockSteps.map((step) => ({
        ...step,
        taskId: crypto.randomUUID(),
        status: step.dependsOn.length === 0 ? 'READY' : 'LOCKED',
      })),
    })),
  }

  mockConsignments.set(consignmentId, consignment)

  return consignment
}

async function mockGetConsignment(id: string): Promise<Consignment | null> {
  return mockConsignments.get(id) || null
}

export async function createConsignment(
  request: CreateConsignmentRequest
): Promise<CreateConsignmentResponse> {
  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 500))
    return mockCreateConsignment(request)
  }

  const response = await fetch(CONSIGNMENT_API_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  })

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return response.json()
}

export async function getConsignment(id: string): Promise<Consignment | null> {
  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 200))
    return mockGetConsignment(id)
  }

  const response = await fetch(`${CONSIGNMENT_API_URL}/${id}`)

  if (!response.ok) {
    if (response.status === 404) {
      return null
    }
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return response.json()
}

export async function getAllConsignments(): Promise<Consignment[]> {
  if (USE_MOCK) {
    // Simulate network delay
    await new Promise((resolve) => setTimeout(resolve, 300))
    // Return consignments sorted by createdAt (newest first)
    return Array.from(mockConsignments.values()).sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )
  }

  const response = await fetch(CONSIGNMENT_API_URL)

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return response.json()
}