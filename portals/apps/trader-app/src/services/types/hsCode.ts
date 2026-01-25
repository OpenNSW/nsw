export interface HSCode {
  id: string
  hsCode: string
  description: string
  category: string
  createdAt: string
  updatedAt: string
}

export interface HSCodeQueryParams {
  hs_code?: string
  limit?: number
  offset?: number
}