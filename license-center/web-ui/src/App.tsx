import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import MainLayout from './layouts/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Licenses from './pages/Licenses'
import Plans from './pages/Plans'
import Alerts from './pages/Alerts'
import Webhooks from './pages/Webhooks'
import Tags from './pages/Tags'
import Logs from './pages/Logs'
import Versions from './pages/Versions'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuthStore()
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <PrivateRoute>
            <MainLayout />
          </PrivateRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="licenses" element={<Licenses />} />
        <Route path="plans" element={<Plans />} />
        <Route path="alerts" element={<Alerts />} />
        <Route path="webhooks" element={<Webhooks />} />
        <Route path="tags" element={<Tags />} />
        <Route path="logs" element={<Logs />} />
        <Route path="versions" element={<Versions />} />
      </Route>
    </Routes>
  )
}

export default App
