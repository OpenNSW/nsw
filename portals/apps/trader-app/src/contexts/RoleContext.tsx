import { createContext, useContext, useMemo, useState, useEffect, type ReactNode } from 'react'
import { useAsgardeo } from '@asgardeo/react'

export type UserRole = 'TRADER' | 'CHA'

function getRoleFromToken(token: string | null | undefined): UserRole {
  if (!token || typeof token !== 'string') return 'TRADER'
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return 'TRADER'
    const payload = parts[1]
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(base64)
    const decoded = JSON.parse(json) as { role?: string }
    if (decoded.role === 'CHA') return 'CHA'
    return 'TRADER'
  } catch {
    return 'TRADER'
  }
}

const RoleContext = createContext<UserRole>('TRADER')

export function RoleProvider({ children }: { children: ReactNode }) {
  const { getAccessToken } = useAsgardeo()
  const [role, setRole] = useState<UserRole>('TRADER')

  useEffect(() => {
    let cancelled = false
    getAccessToken()
      .then((token) => {
        if (!cancelled) setRole(getRoleFromToken(token ?? undefined))
      })
      .catch(() => {
        if (!cancelled) setRole('TRADER')
      })
    return () => { cancelled = true }
  }, [getAccessToken])

  const value = useMemo(() => role, [role])
  return <RoleContext.Provider value={value}>{children}</RoleContext.Provider>
}

export function useRole(): UserRole {
  return useContext(RoleContext)
}
