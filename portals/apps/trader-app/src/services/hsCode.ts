import { apiGet } from './api'
import type { PaginatedResponse } from './types/common'
import type { HSCode, HSCodeQueryParams } from './types/hsCode'



export async function getHSCodes(
  params: HSCodeQueryParams = {}
): Promise<PaginatedResponse<HSCode>> {
  // Convert HSCodeQueryParams to QueryParams
  const queryParams: Record<string, string | number> = {}
  if (params.hsCodeStartsWith) {
    queryParams.hsCodeStartsWith = params.hsCodeStartsWith
  }
  if (params.limit !== undefined) {
    queryParams.limit = params.limit
  }
  if (params.offset !== undefined) {
    queryParams.offset = params.offset
  }

  return apiGet<PaginatedResponse<HSCode>>(
    '/hscodes',
    queryParams
  )
}