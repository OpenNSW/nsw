import { useCallback } from 'react'
import { useAsgardeo } from '@asgardeo/react'

export function useSignOutHandler(): () => void {
  const { signOut } = useAsgardeo()

  return useCallback(() => {
    void (async () => {
      try {
        const signOutResult = await signOut(undefined, (redirectUrl: string) => {
          if (redirectUrl) {
            window.location.assign(redirectUrl)
          }
        })

        if (typeof signOutResult === 'string' && signOutResult) {
          window.location.assign(signOutResult)
        }
      } catch {
        // Let the SDK configuration drive sign-out redirects.
      }
    })()
  }, [signOut])
}