import { createElement, useMemo, type ReactNode } from 'react'
import { useAsgardeo } from '@asgardeo/react'
import { createApiClient } from '../api'
import { ApiContext } from './apiContext'

export function ApiProvider({ children }: { children: ReactNode }) {
  const { getAccessToken } = useAsgardeo()

  const client = useMemo(
    () => createApiClient(async () => getAccessToken()),
    [getAccessToken]
  )

  return createElement(ApiContext.Provider, { value: client }, children)
}
