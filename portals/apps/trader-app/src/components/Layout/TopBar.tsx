import { BellIcon } from '@radix-ui/react-icons'
import { SignedIn, SignedOut, SignInButton, UserDropdown } from '@asgardeo/react'
import { type ReactNode } from 'react'
import { useSignOutHandler } from '../../hooks/useSignOutHandler'
import { RoleSwitcher } from './RoleSwitcher'
import { appConfig, displayName } from '../../config'
import { useProfile } from '../../services/ProfileContext'

function TopBarShell({ children }: { children: ReactNode }) {
  const { profile } = useProfile()

  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-16 bg-white/80 backdrop-blur-md border-b border-gray-200/80 shadow-[0_1px_2px_rgba(15,23,42,0.04)] flex items-center justify-between px-6">
      <div className="flex items-center gap-3 min-w-0">
        {appConfig.branding.systemLogoUrl && (
          <img src={appConfig.branding.systemLogoUrl} alt={displayName} className="h-8 w-auto object-contain" />
        )}
        <span className="text-xl font-bold tracking-tight bg-linear-to-r from-gray-900 to-gray-600 bg-clip-text text-transparent">
          {displayName}
        </span>
        {profile?.company?.name && (
          <div className="flex items-center pl-4 ml-4 border-l border-gray-200 h-6 min-w-0">
            <span
              className="inline-flex items-center gap-2 max-w-72 truncate text-sm font-medium text-gray-700 bg-linear-to-b from-white to-gray-50 border border-gray-200 rounded-full px-3 py-1 select-none shadow-xs"
              title={profile.company.name}
            >
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.15)] shrink-0" />
              <span className="truncate">{profile.company.name}</span>
            </span>
          </div>
        )}
      </div>

      <div className="flex items-center gap-3">{children}</div>
    </header>
  )
}

function TopBarUserActions({ onSignOut, withDivider = true }: { onSignOut: () => void; withDivider?: boolean }) {
  return (
    <div className={`flex items-center gap-3 ${withDivider ? 'pl-3 border-l border-gray-200' : ''}`}>
      <SignedIn>
        <UserDropdown onSignOut={onSignOut} />
      </SignedIn>
      <SignedOut>
        <SignInButton />
      </SignedOut>
    </div>
  )
}

export function TopBar() {
  const handleSignOut = useSignOutHandler()

  return (
    <TopBarShell>
      <RoleSwitcher />
      {/* Notifications */}
      {/* TODO: Show real notifications and link to a notifications page */}
      <button
        type="button"
        aria-label="Notifications"
        className="relative p-2 text-gray-500 hover:text-gray-900 hover:bg-gray-100 rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40"
      >
        <BellIcon className="w-5 h-5" />
        <span className="absolute top-1 right-1 flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75 animate-ping" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-red-500 ring-2 ring-white" />
        </span>
      </button>
      <TopBarUserActions onSignOut={handleSignOut} />
    </TopBarShell>
  )
}
