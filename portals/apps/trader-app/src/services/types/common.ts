
export interface PaginatedResponse<T> {
    items: T[]
    totalCount: number
    offset: number
    limit: number
}
