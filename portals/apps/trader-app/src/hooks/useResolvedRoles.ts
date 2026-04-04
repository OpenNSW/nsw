import { useEffect, useState } from 'react'
import { useAsgardeo } from '@asgardeo/react'
import type { Role } from '../services/RoleContext'
import { mapClaimsToRoles } from '../utils/roleMapper'

interface UseResolvedRolesResult {
  availableRoles: Role[] | null
  isResolvingRoles: boolean
}

export function useResolvedRoles(isSignedIn: boolean): UseResolvedRolesResult {
  const { getDecodedIdToken } = useAsgardeo()
  const [availableRoles, setAvailableRoles] = useState<Role[] | null>(null)
  const [isResolvingRoles, setIsResolvingRoles] = useState(false)

  useEffect(() => {
    let isMounted = true

    const resolveAvailableRoles = async () => {
      if (!isSignedIn) {
        if (isMounted) {
          setAvailableRoles(null)
          setIsResolvingRoles(false)
        }
        return
      }

      setIsResolvingRoles(true)
      try {
        const decodedIdToken = await getDecodedIdToken()
        if (!isMounted) {
          return
        }

        setAvailableRoles(mapClaimsToRoles(decodedIdToken as { groups?: unknown }))
      } catch {
        if (!isMounted) {
          return
        }

        setAvailableRoles([])
      } finally {
        if (isMounted) {
          setIsResolvingRoles(false)
        }
      }
    }

    void resolveAvailableRoles()

    return () => {
      isMounted = false
    }
  }, [getDecodedIdToken, isSignedIn])

  return {
    availableRoles,
    isResolvingRoles,
  }
}
