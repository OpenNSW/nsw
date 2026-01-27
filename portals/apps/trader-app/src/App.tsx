import { Routes, Route } from 'react-router-dom'
import './App.css'
import { Layout } from './components/Layout'
import { DashboardScreen } from "./screens/DashboardScreen.tsx"
import { ConsignmentDetailScreen } from "./screens/ConsignmentDetailScreen.tsx"
import { FormScreen } from "./screens/FormScreen.tsx"

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<DashboardScreen />} />
        <Route path="/consignments" element={<DashboardScreen />} />
        <Route path="/consignments/:consignmentId" element={<ConsignmentDetailScreen />} />
        <Route path="/consignments/:consignmentId/tasks/:taskId" element={<FormScreen />} />
      </Route>
    </Routes>
  )
}

export default App