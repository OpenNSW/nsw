export interface HSCode {
  id: string
  hsCode: string
  description: string
  category: string
}

export interface HSCodeQueryParams {
  hsCodeStartsWith?: string
  limit?: number
  offset?: number
}