import { NavLink } from 'react-router-dom'
import {
  DashboardIcon,
  ArchiveIcon,
  GearIcon,
} from '@radix-ui/react-icons'

interface NavItem {
  name: string
  path: string
  icon: React.ReactNode
}

const mainNavItems: NavItem[] = [
  { name: 'Dashboard', path: '/', icon: <DashboardIcon className="w-5 h-5" /> },
  { name: 'Consignments', path: '/consignments', icon: <ArchiveIcon className="w-5 h-5" /> },
]

const bottomNavItems: NavItem[] = [
  { name: 'Settings', path: '/settings', icon: <GearIcon className="w-5 h-5" /> },
]

export function Sidebar() {
  return (
    <aside className="fixed left-0 top-16 h-[calc(100vh-4rem)] w-56 bg-slate-900 text-white flex flex-col">
      {/* Main Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {mainNavItems.map((item) => (
          <NavLink
            key={item.name}
            to={item.path}
            end={item.path === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-slate-300 hover:bg-slate-800 hover:text-white'
              }`
            }
          >
            {item.icon}
            {item.name}
          </NavLink>
        ))}
      </nav>

      {/* Bottom Navigation */}
      <div className="px-3 py-4 border-t border-slate-700 space-y-1">
        {bottomNavItems.map((item) => (
          <NavLink
            key={item.name}
            to={item.path}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-slate-300 hover:bg-slate-800 hover:text-white'
              }`
            }
          >
            {item.icon}
            {item.name}
          </NavLink>
        ))}
      </div>
    </aside>
  )
}