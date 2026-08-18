import { useEffect, useState } from 'react'
import { rolesApi } from '../services/roles'
import { Dialog, DialogContent } from '../components/ui/dialog'

export function RolesPage() {
  const [roles, setRoles] = useState<Array<{ id: number; name: string; description?: string; permissions?: string[] }>>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [viewing, setViewing] = useState<{ name: string; permissions?: string[] } | null>(null)

  useEffect(() => {
    let active = true
    const run = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const list = await rolesApi.all()
        if (!active) return
        setRoles(list)
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg || 'Failed to load roles')
      } finally {
        setIsLoading(false)
      }
    }
    void run()
    return () => {
      active = false
    }
  }, [])

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Roles</p>
        <h2 className="mt-1 text-2xl font-bold text-slate-900">Role Management</h2>
        <p className="mt-2 text-sm text-slate-600">Read-only list of roles and their permissions.</p>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        {isLoading ? (
          <div className="p-6 text-sm text-slate-500">Loading roles...</div>
        ) : error ? (
          <div className="p-6 text-sm text-red-600">{error}</div>
        ) : roles.length === 0 ? (
          <div className="p-6 text-sm text-slate-500">No roles exposed by backend.</div>
        ) : (
          <table className="min-w-full table-auto">
            <thead>
              <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                <th className="px-3 py-3">Role</th>
                <th className="px-3 py-3">Permissions</th>
                <th className="px-3 py-3">Description</th>
                <th className="px-3 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
                {roles.map((r) => (
                <tr key={r.id || r.name} className="hover:bg-slate-50 transition-colors">
                  <td className="px-3 py-3 text-sm font-medium text-slate-900">{r.name}</td>
                  <td className="px-3 py-3 text-sm text-slate-700">{(r.permissions ?? []).length}</td>
                  <td className="px-3 py-3 text-sm text-slate-700">{r.description ?? ''}</td>
                  <td className="px-3 py-3 text-sm text-right">
                    <button onClick={() => setViewing(r)} className="rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs">View</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {viewing && (
        <Dialog open={Boolean(viewing)} onOpenChange={(open) => { if (!open) setViewing(null) }}>
          <DialogContent>
            <h3 className="text-lg font-semibold">{viewing.name}</h3>
            <div className="mt-3">
              <p className="text-sm text-slate-500">Permissions</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {(viewing.permissions ?? []).length === 0 ? (
                  <span className="text-sm text-slate-400">No permissions information available</span>
                ) : (
                  (viewing.permissions ?? []).map((p) => (
                    <span key={p} className="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">{p}</span>
                  ))
                )}
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

export default RolesPage
