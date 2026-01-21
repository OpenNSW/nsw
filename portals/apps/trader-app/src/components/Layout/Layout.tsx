import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'

export function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      {/* Top Bar - Full Width */}
      <TopBar />

      {/* Sidebar */}
      <Sidebar />

      {/* Main Content Area */}
      <main className="ml-56 pt-16 min-h-screen">
        <div className="h-[calc(100vh-4rem)] overflow-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}