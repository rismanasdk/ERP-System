import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Purchase, CreatePurchaseInput } from '../types/purchase'
import type { Branch } from '../types/auth'
import type { Supplier } from '../types/supplier'
import { purchasesApi } from '../services/purchases'
import { branchesApi } from '../services/branches'
import { suppliersApi } from '../services/suppliers'
import { readStoredAccessToken } from '../services/authSession'
import { CreatePurchaseForm } from '../components/purchasing/CreatePurchaseForm'
import { ApiError } from '../lib/api'
import { useConfirm } from '../utils/confirmUtils'
import { usePagination, PaginationControl } from '../utils/paginationUtils'
import { CreateIcon, SearchIcon } from '../utils/iconsUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { formatDate, formatDateTime } from '../utils/dateUtils'

export function PurchasesPage() {
  const { user } = useAuth()
  const confirmDialog = useConfirm()
  const [items, setItems] = useState<Purchase[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '' })
  const [creating, setCreating] = useState(false)
  const [viewingFor, setViewingFor] = useState<Purchase | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [branchMap, setBranchMap] = useState<Record<number, Branch | null>>({})
  const [supplierMap, setSupplierMap] = useState<Record<number, Supplier | null>>({})

  const token = readStoredAccessToken() ?? undefined
  const rows = useMemo(() => items ?? [], [items])
  const { page, totalPages, pageItems, goToPage, resetPage } = usePagination(rows, 10)
  const branchMapRef = useRef(branchMap)
  const supplierMapRef = useRef(supplierMap)

  useEffect(() => {
    branchMapRef.current = branchMap
  }, [branchMap])

  useEffect(() => {
    supplierMapRef.current = supplierMap
  }, [supplierMap])

  const fetchMissingBranches = useCallback(async (ids: number[]) => {
    const missing = ids.filter((id) => !(id in branchMapRef.current))
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
  }, [token])

  const fetchMissingSuppliers = useCallback(async (ids: number[]) => {
    const missing = ids.filter((id) => !(id in supplierMapRef.current))
    if (!missing.length) return

    const entries = await Promise.all(
      missing.map(async (id) => {
        try {
          const supplier = await suppliersApi.getById(id, token)
          return [id, supplier] as const
        } catch {
          return [id, null] as const
        }
      }),
    )

    setSupplierMap((current) => ({ ...current, ...Object.fromEntries(entries) }))
  }, [token])

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const payload = filter.branch_id ? { branch_id: Number(filter.branch_id) } : undefined
      const res = await purchasesApi.list(payload, token)
      setItems(res)
      resetPage()

      const branchIds = Array.from(new Set(res.map((r) => r.branch_id))).filter(Boolean) as number[]
      const supplierIds = Array.from(new Set(res.map((r) => r.supplier_id))).filter(Boolean) as number[]
      await Promise.all([fetchMissingBranches(branchIds), fetchMissingSuppliers(supplierIds)])
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to purchases.')
        return setError(e.message)
      }
      setError('Unable to load purchases')
    } finally {
      setIsLoading(false)
    }
  }, [fetchMissingBranches, fetchMissingSuppliers, filter.branch_id, resetPage, token])

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

  const onCreate = useCallback(async (payload: CreatePurchaseInput) => {
    setSubmitting(true)
    try {
      await purchasesApi.create(payload, token)
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
    const ok = await confirmDialog({
      title: 'Complete this purchase?',
      description: 'This action will increase inventory for the selected branch.',
      confirmLabel: 'Confirm',
      cancelLabel: 'Cancel',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await purchasesApi.complete(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, token])

  const onCancel = useCallback(async (id: number) => {
    const ok = await confirmDialog({
      title: 'Cancel this purchase?',
      description: 'This may reverse inventory if the purchase was already completed.',
      confirmLabel: 'Cancel',
      cancelLabel: 'Keep it',
      variant: 'destructive',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await purchasesApi.cancel(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, token])

  const isSuperAdmin = user?.roles?.includes('SUPER_ADMIN')
  const canCreate = isSuperAdmin || user?.permissions?.includes('purchases.create')
  const canComplete = isSuperAdmin || user?.permissions?.includes('purchases.complete')
  const canCancel = isSuperAdmin || user?.permissions?.includes('purchases.cancel')

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Purchasing</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Purchases</h2>
          <p className="mt-2 text-sm text-slate-600">Create and manage purchase orders.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button onClick={() => setCreating(true)} className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors">
              <CreateIcon className="h-4 w-4" />
              Create
            </button>
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
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input
              value={filter.branch_id}
              onChange={(e) => setFilter((s) => ({ ...s, branch_id: e.target.value }))}
              placeholder="Filter by branch id"
              className="w-full rounded-md border border-slate-200 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
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
            <div className="p-6 text-slate-600">No purchases found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Number</th>
                  <th className="px-3 py-3">Supplier</th>
                  <th className="px-3 py-3">Branch</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3">Total</th>
                  <th className="px-3 py-3">Created</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {pageItems.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{p.purchase_number}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{supplierMap[p.supplier_id]?.name ?? `#${p.supplier_id}`}</td>
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
                    <td className="px-3 py-3 text-sm text-slate-600">{new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(p.total_amount)}</td>
                    <td className="px-3 py-3 text-sm text-slate-600">{formatDate(p.created_at) ?? '-'}</td>
                    <td className="px-3 py-3 text-right">
                      <div className="inline-flex items-center gap-2">
                        <button onClick={() => setViewingFor(p)} className="rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">View</button>
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

        {!isLoading && rows.length > 0 ? (
          <div className="mt-4 flex justify-center">
            <PaginationControl currentPage={page} totalPages={totalPages} onPageChange={goToPage} />
          </div>
        ) : null}
      </div>

      {creating && (
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-900">Create Purchase</h3>
          <div className="mt-4">
            <CreatePurchaseForm submitting={submitting} onSubmit={onCreate} onCancel={() => setCreating(false)} />
          </div>
        </div>
      )}

      {viewingFor && (
        <Dialog open={Boolean(viewingFor)} onOpenChange={(open) => { if (!open) setViewingFor(null) }}>
          <DialogContent>
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-slate-900">Purchase detail</h3>
            </div>
            <div className="space-y-3">
              <div>
                <div className="text-sm text-slate-500">Number</div>
                <div className="text-sm font-medium">{viewingFor.purchase_number}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Supplier</div>
                <div className="text-sm font-medium">{supplierMap[viewingFor.supplier_id]?.name ?? `#${viewingFor.supplier_id}`}</div>
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
                <div className="text-sm font-medium">
                  {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(viewingFor.total_amount)}
                </div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Created</div>
                <div className="text-sm font-medium">{formatDateTime(viewingFor.created_at) ?? '-'}</div>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

export default PurchasesPage