import {Routes, Route, Navigate} from 'react-router-dom'
import './App.css'
import {Layout} from './components/Layout'
import {ConsignmentScreen} from "./screens/ConsignmentScreen.tsx"
import {ConsignmentDetailScreen} from "./screens/ConsignmentDetailScreen.tsx"
import {TaskDetailScreen} from "./screens/TaskDetailScreen.tsx";
import {PreconsignmentScreen} from "./screens/PreconsignmentScreen.tsx"
import {useAsgardeo, SignedOut} from '@asgardeo/react'
import {LoginScreen} from "./screens/LoginScreen.tsx";

function ProtectedLayout() {
  const {isSignedIn, isLoading} = useAsgardeo()
  if (isLoading) return null
  if (!isSignedIn) return <Navigate to="/login" replace/>
  return <Layout/>
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