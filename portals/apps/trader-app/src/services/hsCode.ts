import type { HSCodePaginatedResponse } from './api'
import type { HSCode, HSCodeQueryParams } from './types/hsCode'

const HS_CODES_API_URL = 'http://localhost:8080/api/hscodes'

export async function getHSCodes(
  params: HSCodeQueryParams = {}
): Promise<HSCodePaginatedResponse<HSCode>> {
  const searchParams = new URLSearchParams()

  if (params.hs_code) {
    searchParams.append('hsCodeStartsWith', params.hs_code)
  }
  if (params.limit !== undefined) {
    searchParams.append('limit', String(params.limit))
  }
  if (params.offset !== undefined) {
    searchParams.append('offset', String(params.offset))
  }

  const queryString = searchParams.toString()
  const url = `${HS_CODES_API_URL}${queryString ? `?${queryString}` : ''}`

  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`)
  }
  return response.json()
}