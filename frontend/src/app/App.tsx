import { Navigate, Route, Routes } from 'react-router-dom'
import { BrowserRouter } from 'react-router-dom'
import { AppShell } from '../components/layout/AppShell'
import { ProtectedRoute } from '../components/auth/ProtectedRoute'
import { AuthProvider } from '../contexts/AuthContext'
import { useAuth } from '../hooks/useAuth'
import { DashboardPage } from '../pages/DashboardPage'
import { LoginPage } from '../pages/LoginPage'
import { PlaceholderPage } from '../pages/PlaceholderPage'
import { ProductsPage } from '../pages/ProductsPage'
import { CustomersPage } from '../pages/CustomersPage'
import { SuppliersPage } from '../pages/SuppliersPage'
import { PurchasesPage } from '../pages/PurchasesPage'
import { InventoryPage } from '../pages/InventoryPage'
import { SalesPage } from '../pages/SalesPage'

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
        <Route path="/products" element={<ProductsPage />} />
        <Route path="/customers" element={<CustomersPage />} />
        <Route path="/suppliers" element={<SuppliersPage />} />
        <Route path="/inventory" element={<InventoryPage />} />
        <Route path="/purchasing" element={<PurchasesPage />} />
        <Route path="/sales" element={<SalesPage />} />
        <Route path="/reports" element={<PlaceholderPage title="Reports" />} />
        <Route path="/app/products" element={<ProductsPage />} />
        <Route path="/app/customers" element={<CustomersPage />} />
        <Route path="/app/suppliers" element={<SuppliersPage />} />
        <Route path="/app/inventory" element={<InventoryPage />} />
        <Route path="/app/purchasing" element={<PurchasesPage />} />
        <Route path="/app/sales" element={<SalesPage />} />
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
