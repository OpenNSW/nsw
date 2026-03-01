import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { WorkflowListScreen } from './screens/WorkflowListScreen'
import { WorkflowDetailScreen } from './screens/WorkflowDetailScreen'
import {appConfig} from "./config.ts";
import {useEffect} from "react";
import { SignedOut, useAsgardeo } from '@asgardeo/react'
import { LoginScreen } from './screens/LoginScreen'
import { setAccessTokenProvider } from './api'

function ProtectedLayout() {
  const { isSignedIn, isLoading, getAccessToken } = useAsgardeo()

  useEffect(() => {
    setAccessTokenProvider(async () => getAccessToken())
    return () => {
      setAccessTokenProvider(null)
    }
  }, [getAccessToken])

  if (isLoading) return null
  if (!isSignedIn) return <Navigate to="/login" replace />
  return <Layout />
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
