export interface HSCode {
  id: string
  hsCode: string
  description: string
  category: string
}

export interface HSCodeListResult {
  totalCount: number
  items: HSCode[]
  offset: number
  limit: number
}

export interface HSCodeQueryParams {
  hsCodeStartsWith?: string
  limit?: number
  offset?: number
}