import { SignedIn, SignedOut, SignInButton } from '@asgardeo/react'
import { Button } from '@radix-ui/themes'
import { UserOnlyTopBar } from '../components/Layout/TopBar'
import { useSignOutHandler } from '../hooks/useSignOutHandler'

export function UnauthorizedScreen() {
  const handleSignOut = useSignOutHandler()

  return (
    <div className="min-h-screen bg-gray-50">
            <UserOnlyTopBar />
            <main className="mt-16 min-h-[calc(100vh-64px)] flex items-center justify-center px-6">
                <div className="w-full max-w-lg rounded-xl border border-gray-200 bg-white p-8 shadow-sm text-center">
                    <div className="mx-auto mb-6 w-fit rounded-full bg-red-50 px-4 py-1 text-sm font-semibold text-red-700">
                        403 Unauthorized
                    </div>

                    <h1 className="text-2xl font-semibold text-gray-900">Access Restricted</h1>
                    <p className="mt-3 text-gray-600">
                        Your account is signed in, but it does not currently have an application role.
                    </p>
                    <div className="mt-8 flex items-center justify-center">
                        <SignedIn>
                            <Button onClick={handleSignOut} size="3" style={{ cursor: 'pointer' }}>
                                Sign out
                            </Button>
                        </SignedIn>
                        <SignedOut>
                            <SignInButton />
                        </SignedOut>
                    </div>
                </div>
            </main>
    </div>
  )
}