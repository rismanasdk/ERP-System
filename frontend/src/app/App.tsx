import { Navigate, Route, Routes } from 'react-router-dom'
import { BrowserRouter } from 'react-router-dom'
import { AppShell } from '../components/layout/AppShell'
import { ProtectedRoute } from '../components/auth/ProtectedRoute'
import { AuthProvider } from '../contexts/AuthContext'
import { useAuth } from '../hooks/useAuth'
import { DashboardPage } from '../pages/DashboardPage'
import { LoginPage } from '../pages/LoginPage'
import { PlaceholderPage } from '../pages/PlaceholderPage'

function AppRoutes() {
  const { isAuthenticated } = useAuth()

  return (
    <Routes>
      <Route path="/login" element={isAuthenticated ? <Navigate to="/dashboard" replace /> : <LoginPage />} />
      <Route
        element={
          <ProtectedRoute>
            <AppShell />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/app/dashboard" element={<DashboardPage />} />
        <Route path="/products" element={<PlaceholderPage title="Products" />} />
        <Route path="/customers" element={<PlaceholderPage title="Customers" />} />
        <Route path="/suppliers" element={<PlaceholderPage title="Suppliers" />} />
        <Route path="/inventory" element={<PlaceholderPage title="Inventory" />} />
        <Route path="/purchasing" element={<PlaceholderPage title="Purchasing" />} />
        <Route path="/sales" element={<PlaceholderPage title="Sales" />} />
        <Route path="/reports" element={<PlaceholderPage title="Reports" />} />
        <Route path="/app/products" element={<PlaceholderPage title="Products" />} />
        <Route path="/app/customers" element={<PlaceholderPage title="Customers" />} />
        <Route path="/app/suppliers" element={<PlaceholderPage title="Suppliers" />} />
        <Route path="/app/inventory" element={<PlaceholderPage title="Inventory" />} />
        <Route path="/app/purchasing" element={<PlaceholderPage title="Purchasing" />} />
        <Route path="/app/sales" element={<PlaceholderPage title="Sales" />} />
        <Route path="/app/reports" element={<PlaceholderPage title="Reports" />} />
      </Route>
      <Route path="*" element={<Navigate to={isAuthenticated ? '/dashboard' : '/login'} replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AuthProvider>
  )
}
