import type {
  Consignment,
  CreateConsignmentRequest,
  CreateConsignmentResponse,
} from './types/consignment'
import type {PaginatedResponse} from "./api.ts";

const CONSIGNMENT_API_URL = 'http://localhost:8080/api/consignments'

export async function createConsignment(
  request: CreateConsignmentRequest
): Promise<CreateConsignmentResponse> {

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

  const response = await fetch(`${CONSIGNMENT_API_URL}/${id}`)

  if (!response.ok) {
    if (response.status === 404) {
      return null
    }
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return response.json()
}

export async function getAllConsignments(): Promise<PaginatedResponse<Consignment>> {

  const response = await fetch(`${CONSIGNMENT_API_URL}?traderId=trader-123`)

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }

  return await response.json()
}