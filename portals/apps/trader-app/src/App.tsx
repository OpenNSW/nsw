import { type ReactNode } from 'react'
import {Routes, Route, Navigate} from 'react-router-dom'
import './App.css'
import {Layout} from './components/Layout'
import {ConsignmentScreen} from "./screens/ConsignmentScreen.tsx"
import {ConsignmentDetailScreen} from "./screens/ConsignmentDetailScreen.tsx"
import {TaskDetailScreen} from "./screens/TaskDetailScreen.tsx";
import {PreconsignmentScreen} from "./screens/PreconsignmentScreen.tsx"
import {SignedOut} from '@asgardeo/react'
import {LoginScreen} from "./screens/LoginScreen.tsx";
import {ApiProvider, useApi} from './services/ApiContext'
import { RoleProvider } from './services/RoleContext'
import { UploadProvider } from '@opennsw/jsonforms-renderers'
import { uploadFile, getDownloadUrl } from './services/upload'
import { useAuthContext } from './hooks/useAuthContext'
import { UnauthorizedScreen } from './screens/UnauthorizedScreen.tsx'

function UploadWrapper({ children }: { children: ReactNode }) {
  const api = useApi()
  return (
    <UploadProvider
      onUpload={(file) => uploadFile(api, file)}
      getDownloadUrl={(key) => getDownloadUrl(api, key)}
    >
      {children}
    </UploadProvider>
  )
}

function ProtectedLayout() {
  const {isSignedIn, isLoading, availableRoles, isResolvingRoles} = useAuthContext()

  if (isLoading || (isSignedIn && (isResolvingRoles || availableRoles === null))) return null
  if (!isSignedIn) return <Navigate to="/login" replace/>
  if (!availableRoles || availableRoles.length === 0) return <UnauthorizedScreen/>
  
  return (
    <ApiProvider>
      <RoleProvider availableGroups={availableRoles} isLoading={isResolvingRoles}>
        <UploadWrapper>
          <Layout/>
        </UploadWrapper>
      </RoleProvider>
    </ApiProvider>
  )
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<SignedOut><LoginScreen/></SignedOut>}/>

      <Route element={<ProtectedLayout/>}>
        <Route path="/" element={<Navigate to="/consignments" replace/>}/>
        <Route path="/consignments" element={<ConsignmentScreen/>}/>
        <Route path="/consignments/:consignmentId" element={<ConsignmentDetailScreen/>}/>
        <Route path="/consignments/:consignmentId/tasks/:taskId" element={<TaskDetailScreen/>}/>
        <Route path="/pre-consignments" element={<PreconsignmentScreen/>}/>
        <Route path="/pre-consignments/:preConsignmentId/tasks/:taskId" element={<TaskDetailScreen/>}/>
      </Route>

      <Route path="*" element={<Navigate to="/login" replace/>}/>
    </Routes>
  )
}

export default App