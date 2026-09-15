import { Navigate, Route, Routes } from 'react-router-dom'
import { BrowserRouter } from 'react-router-dom'
import { AppShell } from '../components/layout/AppShell'
import { ProtectedRoute } from '../components/auth/ProtectedRoute'
import { AuthProvider } from '../contexts/AuthContext'
import { useAuth } from '../hooks/useAuth'
import { DashboardPage } from '../pages/DashboardPage'
import { LoginPage } from '../pages/LoginPage'
import { ProductsPage } from '../pages/ProductsPage'
import { ProductCategoriesPage } from '../pages/ProductCategoriesPage'
import { BranchesPage } from '../pages/BranchesPage'
import { CustomersPage } from '../pages/CustomersPage'
import { SuppliersPage } from '../pages/SuppliersPage'
import { PurchasesPage } from '../pages/PurchasesPage'
import { InventoryPage } from '../pages/InventoryPage'
import { SalesPage } from '../pages/SalesPage'
import { ReportsPage } from '../pages/ReportsPage'
import { OrganizationPage } from '../pages/OrganizationPage'
import { UsersPage } from '../pages/UsersPage'
import { RolesPage } from '../pages/RolesPage'
import { ConfirmDialogProvider } from '../utils/confirmUtils'
import { BranchProvider } from '../contexts/BranchContext'
import { ToastProvider } from '../components/ui/toast'

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
        <Route path="/product-categories" element={<ProductCategoriesPage />} />
        <Route path="/organization" element={<OrganizationPage />} />
        <Route path="/users" element={<UsersPage />} />
          <Route path="/roles" element={<RolesPage />} />
        <Route path="/branches" element={<BranchesPage />} />
        <Route path="/branch" element={<BranchesPage />} />
        <Route path="/customers" element={<CustomersPage />} />
        <Route path="/suppliers" element={<SuppliersPage />} />
        <Route path="/inventory" element={<InventoryPage />} />
        <Route path="/purchasing" element={<PurchasesPage />} />
        <Route path="/sales" element={<SalesPage />} />
        <Route path="/reports" element={<ReportsPage />} />
        <Route path="/app/products" element={<ProductsPage />} />
        <Route path="/app/product-categories" element={<ProductCategoriesPage />} />
        <Route path="/app/organization" element={<OrganizationPage />} />
        <Route path="/app/users" element={<UsersPage />} />
          <Route path="/app/roles" element={<RolesPage />} />
        <Route path="/app/branches" element={<BranchesPage />} />
        <Route path="/app/branch" element={<BranchesPage />} />
        <Route path="/app/customers" element={<CustomersPage />} />
        <Route path="/app/suppliers" element={<SuppliersPage />} />
        <Route path="/app/inventory" element={<InventoryPage />} />
        <Route path="/app/purchasing" element={<PurchasesPage />} />
        <Route path="/app/sales" element={<SalesPage />} />
        <Route path="/app/reports" element={<ReportsPage />} />
      </Route>
      <Route path="*" element={<Navigate to={isAuthenticated ? '/dashboard' : '/login'} replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BranchProvider>
        <ToastProvider>
          <ConfirmDialogProvider>
            <BrowserRouter>
              <AppRoutes />
            </BrowserRouter>
          </ConfirmDialogProvider>
        </ToastProvider>
      </BranchProvider>
    </AuthProvider>
  )
}
