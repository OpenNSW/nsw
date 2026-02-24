import { SignInButton } from '@asgardeo/react'

export function LoginScreen() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-6">
      <div className="w-full max-w-md rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-semibold text-gray-900">Trader Portal</h1>
        <p className="mt-2 text-sm text-gray-600">Sign in to continue to your consignments.</p>
        <div className="mt-6 flex flex-col gap-3">
          <SignInButton />
        </div>
      </div>
    </div>
  )
}
