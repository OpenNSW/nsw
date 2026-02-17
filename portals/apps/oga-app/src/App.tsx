import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { WorkflowListScreen } from './screens/WorkflowListScreen'
import { WorkflowDetailScreen } from './screens/WorkflowDetailScreen'
import {appConfig} from "./config.ts";
import {useEffect} from "react";

function App() {

  useEffect(() => {
    // Set document title
    document.title = appConfig.branding.appName;
  }, []);

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/workflows" replace />} />
        <Route path="/workflows" element={<WorkflowListScreen />} />
        <Route path="/workflows/:workflowId" element={<WorkflowDetailScreen />} />
      </Route>
    </Routes>
  )
}

export default App
