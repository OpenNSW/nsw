import type { ReactNode } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { WorkflowListScreen } from './screens/WorkflowListScreen'
import { WorkflowDetailScreen } from './screens/WorkflowDetailScreen'
import {appConfig} from "./config.ts";
import {useEffect} from "react";
import { SignedOut, useAsgardeo } from '@asgardeo/react'
import { LoginScreen } from './screens/LoginScreen'
import { ApiProvider } from './services/ApiProvider'
import { useApi } from './services/useApi'
import { UploadAuthProvider } from '@opennsw/jsonforms-renderers'

function UploadAuthWrapper({ children }: { children: ReactNode }) {
  const api = useApi()
  return (
    <UploadAuthProvider getAuthHeaders={() => api.getAuthHeaders(false)}>
      {children}
    </UploadAuthProvider>
  )
}

function ProtectedLayout() {
  const { isSignedIn, isLoading } = useAsgardeo()

  if (isLoading) return null
  if (!isSignedIn) return <Navigate to="/login" replace />
  return (
    <ApiProvider>
      <UploadAuthWrapper>
        <Layout />
      </UploadAuthWrapper>
    </ApiProvider>
  )
}

function App() {

  useEffect(() => {
    // Set document title
    document.title = appConfig.branding.appName;
  }, []);

  return (
    <Routes>
      <Route path="/login" element={<SignedOut><LoginScreen /></SignedOut>} />

      <Route element={<ProtectedLayout />}>
        <Route path="/" element={<Navigate to="/workflows" replace />} />
        <Route path="/workflows" element={<WorkflowListScreen />} />
        <Route path="/workflows/:workflowId" element={<WorkflowDetailScreen />} />
      </Route>

      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
