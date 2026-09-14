import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../../hooks/useAuth'
import { LogoutIcon } from '../../utils/iconsUtils'
import { BranchSelector } from './BranchSelector'
import {
  DashboardIcon,
  ProductsIcon,
  CustomersIcon,
  BranchIcon,
  SuppliersIcon,
  InventoryIcon,
  PurchaseIcon,
  SalesIcon,
  ReportsIcon,
  RoleIcon,
  OrganizationIcon,
  UsersIcon,
} from '../../utils/iconsUtils'

// Navigation items. When `permission` is present, the item is shown
// only if the current user has that permission. Do NOT use role names
// (e.g. SUPER_ADMIN) for visibility decisions — the backend is the
// source of truth for permissions.
const navItems = [
  { label: 'Dashboard', to: '/dashboard', icon: DashboardIcon },
  { label: 'Products', to: '/products', icon: ProductsIcon, permission: 'products.read' },
  { label: 'Organization', to: '/organization', icon: OrganizationIcon },
  { label: 'Users', to: '/users', icon: UsersIcon, permission: 'users.read' },
  { label: 'Roles', to: '/roles', icon: RoleIcon, permission: 'roles.read' },
  { label: 'Branch', to: '/branches', icon: BranchIcon },
  { label: 'Customers', to: '/customers', icon: CustomersIcon, permission: 'customers.read' },
  { label: 'Suppliers', to: '/suppliers', icon: SuppliersIcon, permission: 'suppliers.read' },
  { label: 'Inventory', to: '/inventory', icon: InventoryIcon, permission: 'inventory.read' },
  { label: 'Purchasing', to: '/purchasing', icon: PurchaseIcon, permission: 'purchases.read' },
  { label: 'Sales', to: '/sales', icon: SalesIcon, permission: 'sales.read' },
  { label: 'Reports', to: '/reports', icon: ReportsIcon, permission: 'reports.read' },
]

export function AppShell() {
  const { user, logout } = useAuth()

  return (
    <div className="flex min-h-screen bg-slate-100">
      <aside className="w-72 flex flex-col h-screen sticky top-0 bg-slate-900 text-slate-100">
        <div className="border-b border-slate-700 px-6 py-5">
          <div className="flex items-center justify-center gap-2">
            <img src="/ERP-SYSTEM.svg" alt="ERP System logo" className="h-8 w-8" />
            <span className="text-xl uppercase tracking-[0.2em] font-semibold">ERP System</span>
          </div>
        </div>
        
        <nav className="flex-1 space-y-1 px-3 py-4 overflow-y-auto">
          {navItems
            .filter((item) => {
              // If a permission is specified, require it. Otherwise show.
              if (!item.permission) return true
              return Boolean(user?.permissions?.includes(item.permission))
            })
            .map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `flex items-center rounded-lg px-3 py-2 text-sm font-medium transition ${
                    isActive ? 'bg-slate-700 text-white' : 'text-slate-300 hover:bg-slate-800 hover:text-white'
                  }`
                }
              >
                {item.icon && <item.icon className="h-5 w-5" />}
                <span className="ml-3">{item.label}</span>
              </NavLink>
            ))}
        </nav>

        <div className="border-t border-slate-700 p-3 mt-auto">
          <button
            type="button"
            onClick={logout}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-red-600 px-3 py-2.5 text-sm font-medium text-white hover:bg-red-700 transition-colors"
          >
            <LogoutIcon className="h-5 w-5" />
            Logout
          </button>
        </div>
      </aside>

      <div className="flex-1 flex flex-col">
        <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-6 shadow-sm">
          <div>
            <h1 className="text-lg font-semibold text-slate-800">ERP Workspace</h1>
          </div>
          <div className="flex items-center gap-4">
            <BranchSelector />
            <div className="px-3 text-sm font-medium text-slate-700 truncate">
              {user?.name ?? user?.email ?? 'User'}
            </div>
          </div>
        </header>

        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}