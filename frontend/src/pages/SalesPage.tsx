import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Sale, CreateSaleInput } from '../types/sale'
import type { Branch } from '../types/auth'
import { salesApi } from '../services/sales'
import { branchesApi } from '../services/branches'
import { readStoredAccessToken } from '../services/authSession'
import { CreateSaleForm } from '../components/sales/CreateSaleForm'
import { ApiError } from '../lib/api'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { formatDate, formatDateTime } from '../utils/dateUtils'

const currency = (n: number) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)

export function SalesPage() {
  const { user } = useAuth()
  const confirmDialog = useConfirm()
  const [items, setItems] = useState<Sale[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '' })
  const [creating, setCreating] = useState(false)
  const [viewingFor, setViewingFor] = useState<Sale | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [branchMap, setBranchMap] = useState<Record<number, Branch | null>>({})

  const token = readStoredAccessToken() ?? undefined
  const rows = useMemo(() => items ?? [], [items])

  const fetchMissingBranches = useCallback(async (ids: number[]) => {
    const missing = ids.filter((id) => !(id in branchMap))
    if (!missing.length) return

    const entries = await Promise.all(
      missing.map(async (id) => {
        try {
          const branch = await branchesApi.getById(id, token)
          return [id, branch] as const
        } catch {
          return [id, null] as const
        }
      }),
    )

    setBranchMap((current) => ({ ...current, ...Object.fromEntries(entries) }))
  }, [branchMap, token])

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const payload = filter.branch_id ? { branch_id: Number(filter.branch_id) } : undefined
      const res = await salesApi.list(payload, token)
      setItems(res)

      const branchIds = Array.from(new Set(res.map((r) => r.branch_id))).filter(Boolean) as number[]
      await fetchMissingBranches(branchIds)
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to sales.')
        return setError(e.message)
      }
      setError('Unable to load sales')
    } finally {
      setIsLoading(false)
    }
  }, [fetchMissingBranches, filter.branch_id, token])

  useEffect(() => {
    let active = true
    const run = async () => {
      if (!active) return
      await load()
    }
    void run()
    return () => {
      active = false
    }
  }, [load])

  const onCreate = useCallback(async (payload: CreateSaleInput) => {
    setSubmitting(true)
    try {
      await salesApi.create(payload, token)
      setCreating(false)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const onComplete = useCallback(async (id: number) => {
    const current = rows.find((row) => row.id === id)
    if (!current) {
      setError('Sale not found.')
      return
    }
    if (current.status !== 'DRAFT') {
      setError('Only draft sales can be completed.')
      return
    }

    const ok = await confirmDialog({
      title: 'Complete this sale?',
      description: 'This action will decrease inventory for the selected branch.',
      confirmLabel: 'Confirm',
      cancelLabel: 'Cancel',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await salesApi.complete(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, rows, token])

  const onCancel = useCallback(async (id: number) => {
    const current = rows.find((row) => row.id === id)
    if (!current) {
      setError('Sale not found.')
      return
    }
    if (current.status === 'CANCELLED') {
      setError('This sale is already cancelled.')
      return
    }

    const ok = await confirmDialog({
      title: 'Cancel this sale?',
      description: 'This may restore inventory if the sale was already completed.',
      confirmLabel: 'Cancel',
      cancelLabel: 'Keep it',
      variant: 'destructive',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await salesApi.cancel(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, rows, token])

  const onView = useCallback(async (row: Sale) => {
    setViewingFor(row)
  }, [])

  const isSuperAdmin = user?.roles?.includes('SUPER_ADMIN')
  const canCreate = isSuperAdmin || user?.permissions?.includes('sales.create')
  const canComplete = isSuperAdmin || user?.permissions?.includes('sales.complete')
  const canCancel = isSuperAdmin || user?.permissions?.includes('sales.cancel')

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Sales</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Sales</h2>
          <p className="mt-2 text-sm text-slate-600">Create and manage sales orders.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button onClick={() => setCreating(true)} className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors">Create</button>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
          <div className="mt-3">
            <button onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors">Retry</button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <input
              value={filter.branch_id}
              onChange={(e) => setFilter((s) => ({ ...s, branch_id: e.target.value }))}
              placeholder="Filter by branch id"
              className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <button onClick={() => void load()} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-50 transition-colors">Apply</button>
        </div>

        <div className="mt-4 overflow-auto">
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-slate-600">No sales found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Number</th>
                  <th className="px-3 py-3">Branch</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3">Total</th>
                  <th className="px-3 py-3">Created</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rows.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{p.sale_number}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{branchMap[p.branch_id]?.name ?? `#${p.branch_id}`}</td>
                    <td className="px-3 py-3 text-sm">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                        p.status === 'COMPLETED'
                          ? 'bg-green-100 text-green-800'
                          : p.status === 'CANCELLED'
                            ? 'bg-red-100 text-red-800'
                            : 'bg-amber-100 text-amber-800'
                      }`}>
                        {p.status}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-sm text-slate-600">{currency(p.total_amount)}</td>
                    <td className="px-3 py-3 text-sm text-slate-600">{formatDate(p.created_at)}</td>
                    <td className="px-3 py-3 text-right">
                      <div className="inline-flex items-center gap-2">
                        <button onClick={() => void onView(p)} className="rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">View</button>
                        {p.status === 'DRAFT' && canComplete ? (
                          <button onClick={() => void onComplete(p.id)} className="rounded-md border border-green-300 bg-white px-2.5 py-1.5 text-xs font-medium text-green-700 hover:bg-green-50 transition-colors">Complete</button>
                        ) : null}
                        {p.status !== 'CANCELLED' && canCancel ? (
                          <button onClick={() => void onCancel(p.id)} className="rounded-md border border-red-300 bg-white px-2.5 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 transition-colors">Cancel</button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {creating && (
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-900">Create Sale</h3>
          <div className="mt-4">
            <CreateSaleForm submitting={submitting} onSubmit={onCreate} onCancel={() => setCreating(false)} />
          </div>
        </div>
      )}

      {viewingFor && (
        <Dialog open={Boolean(viewingFor)} onOpenChange={(open) => { if (!open) setViewingFor(null) }}>
          <DialogContent>
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-slate-900">Sale detail</h3>
            </div>
            <div className="space-y-3">
              <div>
                <div className="text-sm text-slate-500">Number</div>
                <div className="text-sm font-medium">{viewingFor.sale_number}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Branch</div>
                <div className="text-sm font-medium">{branchMap[viewingFor.branch_id]?.name ?? `#${viewingFor.branch_id}`}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Status</div>
                <div className="text-sm font-medium">{viewingFor.status}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Total</div>
                <div className="text-sm font-medium">{currency(viewingFor.total_amount)}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Created</div>
                <div className="text-sm font-medium">{formatDateTime(viewingFor.created_at)}</div>
              </div>

              {/* NEW: daftar produk */}
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

export default SalesPage