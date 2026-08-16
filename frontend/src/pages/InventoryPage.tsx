import { useCallback, useEffect, useMemo, useState, useRef } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Inventory } from '../types/inventory'
import { inventoryApi } from '../services/inventory'
import { readStoredAccessToken } from '../services/authSession'
import { InventoryAdjustForm } from '../components/inventory/InventoryAdjustForm'
import { InventoryCreateForm } from '../components/inventory/InventoryCreateForm'
import { ApiError } from '../lib/api'
import { productsApi } from '../services/products'
import { branchesApi } from '../services/branches'
import type { Branch } from '../types/auth'
import type { Product } from '../types/product'
import { EditIcon, CreateIcon, CloseIcon, SearchIcon } from '../utils/iconsUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { usePagination, PaginationControl } from '../utils/paginationUtils'

export function InventoryPage() {
  const { user } = useAuth()
  const [items, setItems] = useState<Inventory[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '', product_id: '' })
  const [adjustingFor, setAdjustingFor] = useState<Inventory | null>(null)
  const [viewingFor, setViewingFor] = useState<Inventory | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [productMap, setProductMap] = useState<Record<number, Product | null>>({})
  const [branchMap, setBranchMap] = useState<Record<number, Branch | null>>({})

  const token = readStoredAccessToken() ?? undefined
  const rows = useMemo(() => items, [items])
  const { page, totalPages, pageItems, goToPage, resetPage } = usePagination(rows, 10)

  const productMapRef = useRef(productMap)
  useEffect(() => {
    productMapRef.current = productMap
  }, [productMap])

  const branchMapRef = useRef(branchMap)
  useEffect(() => {
    branchMapRef.current = branchMap
  }, [branchMap])

  const fetchMissingProducts = useCallback(async (ids: number[]) => {
    const missing = ids.filter((id) => !(id in productMapRef.current))
    if (!missing.length) return
    const entries = await Promise.all(
      missing.map(async (id) => {
        try {
          const p = await productsApi.getById(id, token)
          return [id, p] as const
        } catch {
          return [id, null] as const
        }
      }),
    )
    setProductMap((m) => ({ ...m, ...Object.fromEntries(entries) }))
  }, [token])

  const fetchMissingBranches = useCallback(async (ids: number[]) => {
    const missing = ids.filter((id) => !(id in branchMapRef.current))
    if (!missing.length) return
    const entries = await Promise.all(
      missing.map(async (id) => {
        try {
          const b = await branchesApi.getById(id, token)
          return [id, b] as const
        } catch {
          return [id, null] as const
        }
      }),
    )
    setBranchMap((m) => ({ ...m, ...Object.fromEntries(entries) }))
  }, [token])

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const f: { branch_id?: number; product_id?: number } = {}
      if (filter.branch_id) f.branch_id = Number(filter.branch_id)
      if (filter.product_id) f.product_id = Number(filter.product_id)
      const res = await inventoryApi.list(f, token)
      setItems(res)
      resetPage()

      const ids = Array.from(new Set(res.map((r) => r.product_id))).filter(Boolean) as number[]
      const branchIds = Array.from(new Set(res.map((r) => r.branch_id))).filter(Boolean) as number[]
      await Promise.all([fetchMissingProducts(ids), fetchMissingBranches(branchIds)])
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to inventory.')
        return setError(e.message)
      }
      setError('Unable to load inventory')
    } finally {
      setIsLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter, token, fetchMissingProducts, fetchMissingBranches])

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

  const onCreate = useCallback(async (payload: { product_id: number; branch_id: number; quantity: number }) => {
    setSubmitting(true)
    try {
      await inventoryApi.create(payload, token)
      setCreating(false)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const onAdjust = useCallback(async (payload: { movement_type: string; quantity_delta: number; reference_type?: string; reference_id?: number }) => {
    if (!adjustingFor) return
    setSubmitting(true)
    try {
      await inventoryApi.adjust(adjustingFor.id, payload, token)
      setAdjustingFor(null)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [adjustingFor, load, token])

  const canCreate = user?.permissions?.includes('inventory.create')
  const canAdjust = user?.permissions?.includes('inventory.adjust')

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Inventory</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Inventory</h2>
          <p className="mt-2 text-sm text-slate-600">Manage stock levels and movements.</p>
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
            <button onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white">Retry</button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input value={filter.product_id} onChange={(e) => setFilter((s) => ({ ...s, product_id: e.target.value }))} placeholder="Filter by product id" className="w-full rounded-md border border-slate-200 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" />
          </div>
          <input value={filter.branch_id} onChange={(e) => setFilter((s) => ({ ...s, branch_id: e.target.value }))} placeholder="Filter by branch id" className="rounded-md border border-slate-200 px-3 py-2 text-sm" />
          <button onClick={() => void load()} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-50 transition-colors">Apply</button>
        </div>

        <div className="mt-4 overflow-auto">
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-slate-600">No inventory found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Code</th>
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Unit</th>
                  <th className="px-3 py-3">Quantity</th>
                  <th className="px-3 py-3">Branch</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {pageItems.map((i) => {
                  const p = productMap[i.product_id]
                  return (
                    <tr key={i.id} className="hover:bg-slate-50 transition-colors">
                      <td className="px-3 py-3 text-sm font-medium text-slate-900">{p?.sku ?? `#${i.product_id}`}</td>
                      <td className="px-3 py-3 text-sm text-slate-700">{p?.name ?? '-'}</td>
                      <td className="px-3 py-3 text-sm text-slate-500">{p?.unit ?? '-'}</td>
                      <td className="px-3 py-3 text-sm text-slate-700">{i.quantity}</td>
                      <td className="px-3 py-3 text-sm text-slate-500">{branchMap[i.branch_id]?.name ?? i.branch_id ?? '-'}</td>
                      <td className="px-3 py-3 text-sm">
                        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${i.quantity > 0 ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                          {i.quantity > 0 ? 'In Stock' : 'Out of Stock'}
                        </span>
                      </td>
                      <td className="px-3 py-3 text-sm text-right">
                        <div className="inline-flex items-center gap-2">
                          <button onClick={() => setViewingFor(i)} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                            View
                          </button>
                          {canAdjust ? (
                            <button onClick={() => setAdjustingFor(i)} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                              <EditIcon className="h-3.5 w-3.5" />
                              Adjust
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
          {!isLoading && pageItems.length > 0 ? (
            <div className="mt-4 flex justify-center">
              <PaginationControl currentPage={page} totalPages={totalPages} onPageChange={goToPage} />
            </div>
          ) : null}
      </div>

      {creating && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setCreating(false)} />

          <div className="relative w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-slate-900">Create Inventory</h3>
              <button onClick={() => setCreating(false)} className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors">
                <CloseIcon className="h-5 w-5" />
              </button>
            </div>

            <InventoryCreateForm key="create-inventory" submitting={submitting} onSubmit={onCreate} onCancel={() => setCreating(false)} />
          </div>
        </div>
      )}

      {adjustingFor && (
        <Dialog open={Boolean(adjustingFor)} onOpenChange={(open) => { if (!open) setAdjustingFor(null) }}>
          <DialogContent>
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-slate-900">{`Adjust inventory for product ${adjustingFor?.product_id}`}</h3>
            </div>
            <InventoryAdjustForm key={`adjust-${adjustingFor.id}`} initial={{}} submitting={submitting} onSubmit={onAdjust} onCancel={() => setAdjustingFor(null)} />
          </DialogContent>
        </Dialog>
      )}

      {viewingFor && (
        <Dialog open={Boolean(viewingFor)} onOpenChange={(open) => { if (!open) setViewingFor(null) }}>
          <DialogContent>
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-slate-900">Inventory detail</h3>
            </div>
            <div className="space-y-3">
              <div>
                <div className="text-sm text-slate-500">Product</div>
                <div className="text-sm font-medium">{productMap[viewingFor.product_id]?.name ?? `#${viewingFor.product_id}`}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Branch</div>
                <div className="text-sm font-medium">{branchMap[viewingFor.branch_id]?.name ?? viewingFor.branch_id}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Quantity</div>
                <div className="text-sm font-medium">{viewingFor.quantity}</div>
              </div>
              <div>
                <div className="text-sm text-slate-500">Created</div>
                <div className="text-sm font-medium">{viewingFor.created_at ?? '-'}</div>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

export default InventoryPage