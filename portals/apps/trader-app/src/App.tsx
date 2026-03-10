import { type ReactNode } from 'react'
import {Routes, Route, Navigate} from 'react-router-dom'
import './App.css'
import {Layout} from './components/Layout'
import {ConsignmentScreen} from "./screens/ConsignmentScreen.tsx"
import {ConsignmentDetailScreen} from "./screens/ConsignmentDetailScreen.tsx"
import {TaskDetailScreen} from "./screens/TaskDetailScreen.tsx";
import {PreconsignmentScreen} from "./screens/PreconsignmentScreen.tsx"
import {useAsgardeo, SignedOut} from '@asgardeo/react'
import {LoginScreen} from "./screens/LoginScreen.tsx";
import {ApiProvider, useApi} from './services/ApiContext'
import { UploadProvider } from '@opennsw/jsonforms-renderers'
import { uploadFile, getDownloadUrl } from './services/upload'

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
  const {isSignedIn, isLoading} = useAsgardeo()

  if (isLoading) return null
  if (!isSignedIn) return <Navigate to="/login" replace/>
  return (
    <ApiProvider>
      <UploadWrapper>
        <Layout/>
      </UploadWrapper>
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