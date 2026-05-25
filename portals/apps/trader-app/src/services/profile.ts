import { type ApiClient } from './api'

export interface CompanyProfile {
  id: string
  name: string
}

export interface UserProfile {
  id: string
  email: string
  phoneNumber?: string
  data: Record<string, unknown>
  createdAt: string
  updatedAt: string
  company?: CompanyProfile | null
}

export async function getProfile(apiClient: ApiClient): Promise<UserProfile> {
  return apiClient.get<UserProfile>('/profile')
}
