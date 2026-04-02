import {createContext, useContext, useState, type ReactNode} from 'react'

export type Role = 'trader' | 'cha'

interface RoleContextType {
  role: Role
  setRole: (role: Role) => void
  availableRoles: Role[]
  setAvailableRoles: (roles: Role[]) => void
  isLoading: boolean
}

const RoleContext = createContext<RoleContextType | undefined>(undefined)

interface RoleProviderProps {
  children: ReactNode
  availableGroups?: Role[]
  isLoading?: boolean
}

/**
 * Provides global role state for the application.
 * Decoupled from any specific Auth provider.
 */
export function RoleProvider({ children, availableGroups = [], isLoading = false }: RoleProviderProps) {
  const [role, setRoleState] = useState<Role>(() => {
    const savedRole = localStorage.getItem('user-role') as Role
    if (savedRole && availableGroups.includes(savedRole)) {
      return savedRole
    }
    return availableGroups.length > 0 ? availableGroups[0] : 'trader'
  })
  const [availableRoles, setAvailableRolesState] = useState<Role[]>(availableGroups)

  const setRole = (newRole: Role) => {
    if (availableRoles.includes(newRole)) {
      setRoleState(newRole)
      localStorage.setItem('user-role', newRole)
    }
  }

  const setAvailableRoles = (roles: Role[]) => {
    setAvailableRolesState(roles)
    if (roles.length > 0 && !roles.includes(role)) {
      const fallbackRole = roles[0]
      setRoleState(fallbackRole)
      localStorage.setItem('user-role', fallbackRole)
    }
  }

  return (
    <RoleContext.Provider value={{ role, setRole, availableRoles, setAvailableRoles, isLoading }}>
      {children}
    </RoleContext.Provider>
  )
}

export function useRole() {
  const context = useContext(RoleContext)
  if (context === undefined) {
    throw new Error('useRole must be used within a RoleProvider')
  }
  return context
}
