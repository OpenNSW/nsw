import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { useApi } from './ApiContext'
import { getProfile, type UserProfile } from './profile'

interface ProfileContextType {
  profile: UserProfile | null
  isLoading: boolean
  error: Error | null
  refetch: () => Promise<void>
}

const ProfileContext = createContext<ProfileContextType | undefined>(undefined)

export function ProfileProvider({ children }: { children: ReactNode }) {
  const api = useApi()
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchProfile = async () => {
    try {
      setIsLoading(true)
      const data = await getProfile(api)
      setProfile(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void fetchProfile()
  }, [api])

  return (
    <ProfileContext.Provider value={{ profile, isLoading, error, refetch: fetchProfile }}>
      {children}
    </ProfileContext.Provider>
  )
}

export function useProfile() {
  const context = useContext(ProfileContext)
  if (context === undefined) {
    throw new Error('useProfile must be used within a ProfileProvider')
  }
  return context
}
