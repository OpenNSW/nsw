import type { PaginatedResponse } from './common'

// Company is the trimmed projection returned by GET /api/v1/companies.
// Matches the backend `company.Summary` shape (id / name / hasCha) — storage-only fields
// (ou_handle, data, timestamps) are intentionally dropped at the HTTP boundary.
export interface Company {
  id: string
  name: string
  hasCha: boolean
}

export interface CompanyListFilter {
  hasCha?: boolean
  name?: string
}

export type CompanyListResult = PaginatedResponse<Company>
