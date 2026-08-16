import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../../hooks/useAuth'
import { LogoutIcon } from '../../utils/iconsUtils'
import {
  DashboardIcon,
  ProductsIcon,
  CustomersIcon,
  SuppliersIcon,
  InventoryIcon,
  PurchaseIcon,
  SalesIcon,
  ReportsIcon
} from '../../utils/iconsUtils'

const navItems = [
  { label: 'Dashboard', to: '/dashboard', icon: DashboardIcon },
  { label: 'Products', to: '/products', icon: ProductsIcon },
  { label: 'Customers', to: '/customers', icon: CustomersIcon },
  { label: 'Suppliers', to: '/suppliers', icon: SuppliersIcon },
  { label: 'Inventory', to: '/inventory', icon: InventoryIcon },
  { label: 'Purchasing', to: '/purchasing', icon: PurchaseIcon },
  { label: 'Sales', to: '/sales', icon: SalesIcon },
  { label: 'Reports', to: '/reports', icon: ReportsIcon },
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
          {navItems.map((item) => (
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
            <div className="mb-3 px-3 text-sm font-medium text-slate-700 truncate">
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