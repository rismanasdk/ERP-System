import { useAuth } from '../hooks/useAuth'

export function DashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Overview</p>
        <h2 className="mt-3 text-3xl font-bold text-slate-900">Dashboard</h2>
        <p className="mt-2 text-slate-600">Dashboard coming next.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-sm text-slate-500">Authenticated user</p>
          <p className="mt-3 text-lg font-semibold text-slate-900">{user?.name ?? user?.email ?? 'Unknown user'}</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-sm text-slate-500">Current branch</p>
          <p className="mt-3 text-lg font-semibold text-slate-900">Not yet selected</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-sm text-slate-500">Quick navigation</p>
          <p className="mt-3 text-lg font-semibold text-slate-900">Products, Sales, Reports</p>
        </div>
      </div>
    </div>
  )
}
