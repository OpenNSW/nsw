import type { Company, CompanyListFilter, CompanyListResult } from './types/company'
import { defaultApiClient, type ApiClient, type QueryParams } from './api'

// getCompanies queries GET /api/v1/companies with optional has_cha and name filters.
// Returns the items list directly — callers that need pagination metadata can switch to
// getCompaniesResult below once the backend implements offset/limit.
export async function getCompanies(
  filter: CompanyListFilter = {},
  apiClient: ApiClient = defaultApiClient,
): Promise<Company[]> {
  const result = await getCompaniesResult(filter, apiClient)
  return result.items
}

export async function getCompaniesResult(
  filter: CompanyListFilter = {},
  apiClient: ApiClient = defaultApiClient,
): Promise<CompanyListResult> {
  const params: QueryParams = {}
  if (filter.hasCha !== undefined) params.has_cha = String(filter.hasCha)
  if (filter.name && filter.name.trim()) params.name = filter.name.trim()
  return apiClient.get<CompanyListResult>('/companies', params)
}
