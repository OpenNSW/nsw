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

// CompanyListResult mirrors the backend `company.ListResult` envelope.
// The envelope is pagination-shaped from day one even though the server currently returns
// the full list — adding offset/limit later is non-breaking. Note the field is `total`
// (not `totalCount`) to match the backend contract.
export interface CompanyListResult {
  items: Company[]
  total: number
  offset: number
  limit: number
}
