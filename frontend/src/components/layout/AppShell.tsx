import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../../hooks/useAuth'

const navItems = [
  { label: 'Dashboard', to: '/dashboard' },
  { label: 'Products', to: '/products' },
  { label: 'Customers', to: '/customers' },
  { label: 'Suppliers', to: '/suppliers' },
  { label: 'Inventory', to: '/inventory' },
  { label: 'Purchasing', to: '/purchasing' },
  { label: 'Sales', to: '/sales' },
  { label: 'Reports', to: '/reports' },
]

export function AppShell() {
  const { user, logout } = useAuth()

  return (
    <div className="flex min-h-screen bg-slate-100">
      <aside className="w-72 bg-slate-900 text-slate-100">
        <div className="border-b border-slate-700 px-6 py-5">
          <div className="text-xs uppercase tracking-[0.2em] text-slate-400">ERP</div>
          <div className="mt-2 text-xl font-semibold">System</div>
        </div>
        <nav className="space-y-1 px-3 py-4">
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
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="flex-1">
        <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-4 shadow-sm">
          <div>
            <h1 className="text-lg font-semibold text-slate-800">ERP Workspace</h1>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-sm text-slate-600">
              {user?.name ?? user?.email ?? 'User'}
            </div>
            <button
              type="button"
              onClick={logout}
              className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            >
              Logout
            </button>
          </div>
        </header>

        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
