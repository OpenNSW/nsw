import { useContext } from 'react'
import type { ApiClient } from '../api'
import { ApiContext } from './apiContext'

export function useApi(): ApiClient {
  const apiClient = useContext(ApiContext)
  if (!apiClient) {
    throw new Error('useApi must be used within ApiProvider')
  }
  return apiClient
}
