import type { ReactNode } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { useAuth } from './auth'
import Layout from './components/Layout'
import Brokerage from './pages/Brokerage'
import ChangePassword from './pages/ChangePassword'
import Dashboard from './pages/Dashboard'
import Dividends from './pages/Dividends'
import Funds from './pages/Funds'
import Insurance from './pages/Insurance'
import Login from './pages/Login'
import Pvd from './pages/Pvd'
import SalaryTax from './pages/SalaryTax'
import Savings from './pages/Savings'

function RequireAuth({ children, allowPending = false }: { children: ReactNode; allowPending?: boolean }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <div className="center muted">Loading…</div>
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (user.mustChangePassword && !allowPending) return <Navigate to="/change-password" replace />
  return children
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/change-password"
        element={
          <RequireAuth allowPending>
            <ChangePassword />
          </RequireAuth>
        }
      />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="brokerage" element={<Brokerage />} />
        <Route path="funds" element={<Funds />} />
        <Route path="pvd" element={<Pvd />} />
        <Route path="savings" element={<Savings />} />
        <Route path="insurance" element={<Insurance />} />
        <Route path="dividends" element={<Dividends />} />
        <Route path="salary" element={<SalaryTax />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
