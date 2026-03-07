import { createContext, useContext, useMemo, useState, useEffect, type ReactNode } from 'react'
import { useAsgardeo } from '@asgardeo/react';
import { jwtDecode } from 'jwt-decode';

export type UserRole = 'TRADER' | 'CHA'

function getRoleFromToken(token: string | null | undefined): UserRole {
  if (!token) {
    return 'TRADER'
  }
  try {
    const decoded = jwtDecode<{ role?: string }>(token)
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
