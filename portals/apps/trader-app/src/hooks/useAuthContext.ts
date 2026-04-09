import { useEffect, useState } from 'react'
import { useAsgardeo } from '@asgardeo/react'
import type { Role } from '../services/RoleContext'
import { mapClaimsToRoles } from '../utils/roleMapper'

interface UseAuthContextResult {
  isSignedIn: boolean
  isLoading: boolean
  availableRoles: Role[] | null
  isResolvingRoles: boolean
}

export function useAuthContext(): UseAuthContextResult {
  const { isSignedIn, isLoading, getDecodedIdToken } = useAsgardeo()
  const [availableRoles, setAvailableRoles] = useState<Role[] | null>(null)
  const [isResolvingRoles, setIsResolvingRoles] = useState(false)

  useEffect(() => {
    let isMounted = true

    const resolveAvailableRoles = async () => {
      if (isLoading) {
        return
      }

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

        const claimsCandidate =
          (decodedIdToken as { decodedIDTokenPayload?: unknown })?.decodedIDTokenPayload ??
          (decodedIdToken as { payload?: unknown })?.payload ??
          decodedIdToken

        setAvailableRoles(mapClaimsToRoles(claimsCandidate as { groups?: unknown }))
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
  }, [getDecodedIdToken, isLoading, isSignedIn])

  return {
    isSignedIn,
    isLoading,
    availableRoles,
    isResolvingRoles,
  }
}