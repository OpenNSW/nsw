import {BellIcon, BackpackIcon, IdCardIcon} from '@radix-ui/react-icons'
import {SignedIn, SignedOut, SignInButton, UserDropdown, useAsgardeo} from '@asgardeo/react'
import {Select, Flex, Text, Box} from '@radix-ui/themes'
import {useRole, type Role} from '../../services/RoleContext'
import {useCallback, type ReactNode} from 'react'

const ROLE_CONFIG: Record<Role, {
  label: string;
  description: string;
  dropdownDescription: string;
  icon: ReactNode
}> = {
  'trader': {
    label: 'Trader',
    description: 'Managing consignments',
    dropdownDescription: 'Create and manage consignments',
    icon: <BackpackIcon className="text-blue-600"/>,
  },
  'cha': {
    label: 'CHA',
    description: 'Handling Customs Clearances',
    dropdownDescription: 'Handle customs clearances',
    icon: <IdCardIcon className="text-orange-600"/>,
  },
}

function RoleDisplay({role, showPrimaryLabel}: { role: Role; showPrimaryLabel: boolean }) {
  const {label, description, icon} = ROLE_CONFIG[role]
  return (
    <Flex align="center" gap="3" className="w-60 text-left">
      <Box className="p-1.5 bg-white rounded-md shadow-sm border border-gray-100">
        {icon}
      </Box>
      <Box className="flex-1">
        <Flex align="center" gap="1">
          <Text size="1" weight="bold" className="block leading-none">{label}</Text>
          {showPrimaryLabel && (
            <Text size="1" color="gray" className="font-normal">(Primary)</Text>
          )}
        </Flex>
        <Text size="1" color="gray" className="block leading-tight mt-0.5">{description}</Text>
      </Box>
    </Flex>
  )
}

function useSignOutHandler(): () => void {
  const {signOut} = useAsgardeo()

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

function TopBarShell({ children }: { children: ReactNode }) {
  return (
    <header
      className="fixed top-0 left-0 right-0 z-50 h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
      <div className="flex items-center">
        <span className="text-xl font-bold text-gray-900">Trader Portal</span>
      </div>

      <div className="flex items-center gap-4">
        {children}
      </div>
    </header>
  )
}

function TopBarUserActions({ onSignOut, withDivider = true }: { onSignOut: () => void; withDivider?: boolean }) {
  return (
    <div className={`flex items-center gap-3 ${withDivider ? 'pl-3 border-l border-gray-200' : ''}`}>
      <SignedIn>
        <UserDropdown onSignOut={onSignOut}/>
      </SignedIn>
      <SignedOut>
        <SignInButton/>
      </SignedOut>
    </div>
  )
}

export function UserOnlyTopBar() {
  const handleSignOut = useSignOutHandler()

  return (
    <TopBarShell>
      <TopBarUserActions onSignOut={handleSignOut} withDivider={false}/>
    </TopBarShell>
  )
}

export function TopBar() {
  const handleSignOut = useSignOutHandler()
  const {role, setRole, availableRoles, isLoading} = useRole()

  const showSwitcher = availableRoles.length > 1
  return (
    <TopBarShell>
        {/* Role Switcher (Dynamic based on user permissions) */}
        <Box className="flex-1 max-w-md px-8">
          {!isLoading ? (
            <Box className="w-full h-full">
              <Select.Root
                value={role}
                onValueChange={(val) => setRole(val as Role)}
                disabled={!showSwitcher}
              >
                <Select.Trigger
                  variant="ghost"
                  className={`w-full h-12 p-4 transition-all ${showSwitcher ? 'cursor-pointer hover:bg-gray-100' : 'cursor-default'}`}
                >
                  <RoleDisplay role={role} showPrimaryLabel={!showSwitcher}/>
                </Select.Trigger>

                {showSwitcher && (
                  <Select.Content position="popper" className="w-full min-w-[320px]">
                    {availableRoles.map((r) => {
                      const {label, dropdownDescription, icon} = ROLE_CONFIG[r]
                      return (
                        <Select.Item 
                          key={r} 
                          value={r}
                          className="py-2 focus:bg-gray-100 data-highlighted:bg-gray-100! data-highlighted:text-inherit! cursor-pointer transition-colors"
                        >

                          <Flex direction="column" py="1">
                            <Flex align="center" gap="2">
                              {icon}
                              <Text weight="bold" size="1" className="text-gray-900">{label}</Text>
                            </Flex>
                            <Text size="1" color="gray">{dropdownDescription}</Text>
                          </Flex>
                        </Select.Item>
                      )
                    })}
                  </Select.Content>
                )}
              </Select.Root>
            </Box>
          ) : (
            <Box className="w-full h-12 bg-gray-50 animate-pulse rounded-lg border border-gray-100"/>
          )}
        </Box>
        {/* Notifications */}
        <button
          className="relative p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors">
          <BellIcon className="w-5 h-5"/>
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-red-500 rounded-full"></span>
        </button>

        <TopBarUserActions onSignOut={handleSignOut}/>
    </TopBarShell>
  )
}
