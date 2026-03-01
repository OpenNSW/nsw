import { createContext, useContext, useEffect, useMemo, useRef, type ReactNode } from 'react'
import { useAsgardeo } from '@asgardeo/react'
import { createApiClient, type ApiClient } from './api'

const ApiContext = createContext<ApiClient | null>(null)

export function ApiProvider({ children }: { children: ReactNode }) {
  const { getAccessToken } = useAsgardeo()
  const getAccessTokenRef = useRef(getAccessToken)

  useEffect(() => {
    getAccessTokenRef.current = getAccessToken
  }, [getAccessToken])

  const client = useMemo(
    () => createApiClient(async () => getAccessTokenRef.current()),
    []
  )

  return <ApiContext.Provider value={client}>{children}</ApiContext.Provider>
}

export function useApi(): ApiClient {
  const apiClient = useContext(ApiContext)
  if (!apiClient) {
    throw new Error('useApi must be used within ApiProvider')
  }
  return apiClient
}
