import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar, UserOnlyTopBar } from './TopBar'
import {type ReactNode, useState} from "react";

interface LayoutProps {
  children?: ReactNode
  showSidebar?: boolean
  topBarMode?: 'full' | 'user-only'
}

export function Layout({ children, showSidebar = true, topBarMode = 'full' }: LayoutProps) {
  const [isSidebarExpanded, setIsSidebarExpanded] = useState(() => {
    const savedState = localStorage.getItem('sidebarExpanded');
    // Default to true if no saved state is found
    return savedState !== null ? savedState === 'true' : true;
  });

  const sidebarWidth = showSidebar && isSidebarExpanded ? 256 : 80; // w-64 = 256px, w-20 = 80px
  // Save sidebar state to localStorage when it changes
  const handleToggleSidebar = () => {
    setIsSidebarExpanded((prev) => {
      const newState = !prev;
      localStorage.setItem('sidebarExpanded', String(newState));
      return newState;
    });
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {topBarMode === 'user-only' ? <UserOnlyTopBar /> : <TopBar />}

      <div className="flex">
      {showSidebar && (
        <Sidebar isExpanded={isSidebarExpanded} onToggle={handleToggleSidebar} />
      )}

      <main
        style={
          showSidebar
            ? { marginLeft: `${sidebarWidth}px`, width: `calc(100% - ${sidebarWidth}px)` }
            : { width: '100%' }
        }
        className="min-h-[calc(100vh-64px)] transition-all duration-300 mt-16"
      >
          {children ?? <Outlet />}
      </main>
      </div>
    </div>
  )
}