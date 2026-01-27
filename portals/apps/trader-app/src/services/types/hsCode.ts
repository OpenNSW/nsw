export interface HSCode {
  id: string
  hsCode: string
  description: string
  category: string
  createdAt: string
  updatedAt: string
}

export interface HSCodeQueryParams {
  hsCodeStartsWith?: string
  limit?: number
  offset?: number
}