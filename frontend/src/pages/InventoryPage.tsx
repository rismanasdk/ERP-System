import { useCallback, useEffect, useMemo, useState, useRef } from 'react'
import { useAuth } from '../hooks/useAuth'
import { useBranch } from '../contexts/BranchContext'
import type { Inventory, StockMovement, StockTransfer } from '../types/inventory'
import { inventoryApi } from '../services/inventory'
import { readStoredAccessToken } from '../services/authSession'
import { InventoryAdjustForm } from '../components/inventory/InventoryAdjustForm'
import { InventoryCreateForm } from '../components/inventory/InventoryCreateForm'
import { ApiError } from '../lib/api'
import { productsApi } from '../services/products'
import { branchesApi } from '../services/branches'
import type { Branch } from '../types/auth'
import type { Product } from '../types/product'
import { EditIcon, CreateIcon, CloseIcon, SearchIcon, ViewIcon } from '../utils/iconsUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { useToast } from '../components/ui/toast'
import { usePagination, PaginationControl } from '../utils/paginationUtils'

function getAdjustErrorToast(error: unknown, payload: { movement_type: string; quantity_delta: number }, inventory: Inventory) {
  if (error instanceof ApiError && error.message.toLowerCase().includes('insufficient stock')) {
    const outgoingQuantity = Math.abs(payload.quantity_delta)
    return {
      title: 'Stok tidak mencukupi',
      description: `Stok tersedia ${inventory.quantity}, sementara quantity keluar ${outgoingQuantity}.`,
    }
  }

  return {
    title: 'Gagal menyesuaikan stok',
    description: error instanceof ApiError ? error.message : 'Terjadi kesalahan saat menyimpan perubahan stok.',
  }
}

export function InventoryPage() {
  const { user } = useAuth()
  const { selectedBranch, isAllBranches, accessibleBranches } = useBranch()
  const { toast } = useToast()
  const [items, setItems] = useState<Inventory[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '', product_id: '' })
  const [adjustingFor, setAdjustingFor] = useState<Inventory | null>(null)
  const [viewingFor, setViewingFor] = useState<Inventory | null>(null)
  const [history, setHistory] = useState<StockMovement[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [productMap, setProductMap] = useState<Record<number, Product | null>>({})
  const [branchMap, setBranchMap] = useState<Record<number, Branch | null>>({})
  const [transfers, setTransfers] = useState<StockTransfer[]>([])
  const [transferOpen, setTransferOpen] = useState(false)
  const [transferForm, setTransferForm] = useState({ source_branch_id: '', destination_branch_id: '', product_id: '', quantity: '', notes: '' })
  const [transferError, setTransferError] = useState<string | null>(null)
  const [transferDetail, setTransferDetail] = useState<StockTransfer | null>(null)
  const [transferDetailLoading, setTransferDetailLoading] = useState(false)

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
      if (selectedBranch && selectedBranch.id > 0 && !isAllBranches) f.branch_id = selectedBranch.id
      if (filter.branch_id) f.branch_id = Number(filter.branch_id)
      if (filter.product_id) f.product_id = Number(filter.product_id)
      const res = await inventoryApi.list(f, token)
      setItems(res)
      resetPage()
      const transferItems = await inventoryApi.listTransfers(token)
      setTransfers(Array.isArray(transferItems) ? transferItems : [])

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
  }, [filter, token, fetchMissingProducts, fetchMissingBranches, selectedBranch, isAllBranches])

  const submitTransfer = useCallback(async () => {
    const sourceBranchID = Number(transferForm.source_branch_id)
    const destinationBranchID = Number(transferForm.destination_branch_id)
    const productID = Number(transferForm.product_id)
    const quantity = Number(transferForm.quantity)
    const sourceStock = items.find((item) => item.branch_id === sourceBranchID && item.product_id === productID)?.quantity ?? 0
    if (!sourceBranchID || !destinationBranchID || !productID || quantity <= 0) {
      setTransferError('Source, destination, product, and a quantity greater than zero are required.')
      return
    }
    if (sourceBranchID === destinationBranchID) {
      setTransferError('Source and destination branches must be different.')
      return
    }
    if (quantity > sourceStock) {
      setTransferError(`Insufficient stock. Available: ${sourceStock}.`)
      return
    }
    setSubmitting(true)
    setTransferError(null)
    try {
      await inventoryApi.createTransfer({ source_branch_id: sourceBranchID, destination_branch_id: destinationBranchID, product_id: productID, quantity, notes: transferForm.notes || undefined }, token)
      setTransferOpen(false)
      setTransferForm({ source_branch_id: '', destination_branch_id: '', product_id: '', quantity: '', notes: '' })
      await load()
    } catch (err) {
      setTransferError(err instanceof ApiError ? err.message : 'Unable to create stock transfer.')
    } finally {
      setSubmitting(false)
    }
  }, [items, load, token, transferForm])

  const openTransferDetail = useCallback(async (id: number) => {
    setTransferDetailLoading(true)
    try {
      setTransferDetail(await inventoryApi.getTransfer(id, token))
    } catch {
      setTransferDetail(null)
    } finally {
      setTransferDetailLoading(false)
    }
  }, [token])
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
      const errorToast = getAdjustErrorToast(err, payload, adjustingFor)
      toast({
        ...errorToast,
        variant: 'destructive',
      })
    } finally {
      setSubmitting(false)
    }
  }, [adjustingFor, load, toast, token])

  const loadHistory = useCallback(async (inventory: Inventory | null) => {
    if (!inventory) {
      setHistory([])
      return
    }

    setHistoryLoading(true)
    try {
      const res = await inventoryApi.listMovements({
        branch_id: inventory.branch_id,
        product_id: inventory.product_id,
      }, token)
      setHistory(Array.isArray(res) ? res : [])
    } catch {
      setHistory([])
    } finally {
      setHistoryLoading(false)
    }
  }, [token])

  const canRead = user ? Boolean(user?.permissions?.includes('inventory.read')) : true

  useEffect(() => {
    let active = true
    const run = async () => {
      if (!active) return
      if (!canRead) {
        setIsLoading(false)
        setError(null)
        return
      }
      await load()
    }
    void run()
    return () => {
      active = false
    }
  }, [load, canRead])

  if (!canRead) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">You do not have permission to view inventory.</div>
    )
  }

  const canCreate = user?.permissions?.includes('inventory.create')
  const canAdjust = user?.permissions?.includes('inventory.adjust')
  const transferSourceBranchID = Number(transferForm.source_branch_id)
  const transferProductID = Number(transferForm.product_id)
  const currentSourceStock = items.find((item) => item.branch_id === transferSourceBranchID && item.product_id === transferProductID)?.quantity ?? 0
  const transferProducts = Array.from(new Map(items.filter((item) => !transferSourceBranchID || item.branch_id === transferSourceBranchID).map((item) => [item.product_id, item])).values())

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Inventory</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Inventory</h2>
          <p className="mt-2 text-sm text-slate-600">Manage stock levels and movements.</p>
        </div>
        <div className="flex items-center gap-3">
          {canAdjust ? (
            <button onClick={() => { setTransferOpen(true); setTransferError(null) }} className="inline-flex items-center gap-2 rounded-md border border-indigo-200 bg-indigo-50 px-4 py-2 text-sm font-medium text-indigo-700 hover:bg-indigo-100 transition-colors">
              <CreateIcon className="h-4 w-4" />
              Stock Transfer
            </button>
          ) : null}
          {canCreate ? (
            <button onClick={() => setCreating(true)} className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors">
              <CreateIcon className="h-4 w-4" />
              Create
            </button>
          ) : null}
        </div>
      </div>

      {transferOpen ? (
        <Dialog open={transferOpen} onOpenChange={setTransferOpen}>
          <DialogContent>
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-slate-900">Stock Transfer</h3>
            </div>
            <div className="space-y-4">
              <label className="block text-sm font-medium text-slate-700">Source Branch
                <select value={transferForm.source_branch_id} onChange={(e) => setTransferForm((form) => ({ ...form, source_branch_id: e.target.value, product_id: '' }))} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 text-sm">
                  <option value="">Select source branch</option>
                  {accessibleBranches.map((branch) => <option key={branch.id} value={branch.id}>{branch.name}</option>)}
                </select>
              </label>
              <label className="block text-sm font-medium text-slate-700">Destination Branch
                <select value={transferForm.destination_branch_id} onChange={(e) => setTransferForm((form) => ({ ...form, destination_branch_id: e.target.value }))} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 text-sm">
                  <option value="">Select destination branch</option>
                  {accessibleBranches.map((branch) => <option key={branch.id} value={branch.id}>{branch.name}</option>)}
                </select>
              </label>
              <label className="block text-sm font-medium text-slate-700">Product
                <select value={transferForm.product_id} onChange={(e) => setTransferForm((form) => ({ ...form, product_id: e.target.value }))} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 text-sm">
                  <option value="">Select product</option>
                  {transferProducts.map((item) => <option key={item.product_id} value={item.product_id}>{productMap[item.product_id]?.name ?? `#${item.product_id}`}</option>)}
                </select>
              </label>
              <div className="rounded-md bg-slate-50 px-3 py-2 text-sm text-slate-600">Current Stock: <span className="font-semibold text-slate-900">{currentSourceStock}</span></div>
              <label className="block text-sm font-medium text-slate-700">Quantity
                <input type="number" min="1" value={transferForm.quantity} onChange={(e) => setTransferForm((form) => ({ ...form, quantity: e.target.value }))} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 text-sm" />
              </label>
              <label className="block text-sm font-medium text-slate-700">Notes
                <textarea value={transferForm.notes} onChange={(e) => setTransferForm((form) => ({ ...form, notes: e.target.value }))} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 text-sm" rows={3} />
              </label>
              {transferError ? <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{transferError}</div> : null}
              <div className="flex justify-end gap-2">
                <button onClick={() => setTransferOpen(false)} className="rounded-md border border-slate-200 px-4 py-2 text-sm">Cancel</button>
                <button onClick={() => void submitTransfer()} disabled={submitting} className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">{submitting ? 'Transferring...' : 'Transfer'}</button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      ) : null}

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
                          <button onClick={() => { setViewingFor(i); void loadHistory(i) }} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                            <ViewIcon className="h-3.5 w-3.5" />
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

      {transfers.length > 0 ? (
        <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <h3 className="mb-4 text-lg font-semibold text-slate-900">Transfer History</h3>
          <div className="overflow-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="border-b border-slate-100 text-xs font-medium uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-3 py-3">Transfer</th>
                  <th className="px-3 py-3">Date</th>
                  <th className="px-3 py-3">Product</th>
                  <th className="px-3 py-3">From</th>
                  <th className="px-3 py-3">To</th>
                  <th className="px-3 py-3">Quantity</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3">Created By</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {transfers.map((transfer) => (
                  <tr key={transfer.id}>
                    <td className="px-3 py-3"><button onClick={() => void openTransferDetail(transfer.id)} className="font-medium text-indigo-700 hover:text-indigo-900">#{transfer.id}</button></td>
                    <td className="px-3 py-3 text-slate-600">{transfer.created_at ? new Date(transfer.created_at).toLocaleString() : '-'}</td>
                    <td className="px-3 py-3 font-medium text-slate-900">{productMap[transfer.product_id]?.name ?? `#${transfer.product_id}`}</td>
                    <td className="px-3 py-3 text-slate-600">{transfer.source_branch_name ?? branchMap[transfer.source_branch_id]?.name ?? `#${transfer.source_branch_id}`}</td>
                    <td className="px-3 py-3 text-slate-600">{transfer.destination_branch_name ?? branchMap[transfer.destination_branch_id]?.name ?? `#${transfer.destination_branch_id}`}</td>
                    <td className="px-3 py-3 text-slate-600">{transfer.quantity}</td>
                    <td className="px-3 py-3"><span className="inline-flex rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">{transfer.status}</span></td>
                    <td className="px-3 py-3 text-slate-600">{transfer.created_by_name ?? 'Unknown User'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {(transferDetailLoading || transferDetail) ? (
        <Dialog open={transferDetailLoading || Boolean(transferDetail)} onOpenChange={(open) => { if (!open) setTransferDetail(null) }}>
          <DialogContent>
            {transferDetailLoading ? <div className="py-6 text-sm text-slate-500">Loading transfer...</div> : transferDetail ? (
              <div className="space-y-4">
                <h3 className="text-lg font-semibold text-slate-900">Stock Transfer #{transferDetail.id}</h3>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div><div className="text-slate-500">Product</div><div className="font-medium">{productMap[transferDetail.product_id]?.name ?? `#${transferDetail.product_id}`}</div></div>
                  <div><div className="text-slate-500">Status</div><span className="inline-flex rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">{transferDetail.status}</span></div>
                  <div><div className="text-slate-500">From</div><div className="font-medium">{transferDetail.source_branch_name ?? `#${transferDetail.source_branch_id}`}</div></div>
                  <div><div className="text-slate-500">To</div><div className="font-medium">{transferDetail.destination_branch_name ?? `#${transferDetail.destination_branch_id}`}</div></div>
                  <div><div className="text-slate-500">Quantity</div><div className="font-medium">{transferDetail.quantity}</div></div>
                  <div><div className="text-slate-500">Created By</div><div className="font-medium">{transferDetail.created_by_name || 'Unknown User'}</div></div>
                </div>
                <div className="border-t border-slate-200 pt-4 text-sm">
                  <div className="mb-2 font-semibold text-slate-700">Movement</div>
                  <div className="flex justify-between"><span>{transferDetail.source_branch_name ?? `#${transferDetail.source_branch_id}`}</span><span className="font-medium text-rose-700">-{transferDetail.quantity}</span></div>
                  <div className="flex justify-between"><span>{transferDetail.destination_branch_name ?? `#${transferDetail.destination_branch_id}`}</span><span className="font-medium text-emerald-700">+{transferDetail.quantity}</span></div>
                </div>
                <div className="text-sm"><div className="text-slate-500">Notes</div><div>{transferDetail.notes || '-'}</div></div>
              </div>
            ) : null}
          </DialogContent>
        </Dialog>
      ) : null}

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

              <div className="pt-4 border-t border-slate-200">
                <div className="mb-2 text-sm font-semibold text-slate-700">Stock Movement History</div>

                {historyLoading ? (
                  <div className="space-y-2">
                    <div className="h-8 w-full rounded bg-slate-200" />
                    <div className="h-8 w-3/4 rounded bg-slate-200" />
                  </div>
                ) : history.length === 0 ? (
                  <div className="rounded-md border border-dashed border-slate-200 bg-slate-50 p-3 text-sm text-slate-500">
                    No stock movement history available.
                  </div>
                ) : (
                  <div className="max-h-72 overflow-auto rounded-md border border-slate-200">
                    <table className="min-w-full text-left text-xs">
                      <thead className="bg-slate-50 text-slate-500">
                        <tr>
                          <th className="px-2 py-2">Date</th>
                          <th className="px-2 py-2">Type</th>
                          <th className="px-2 py-2">Qty</th>
                          <th className="px-2 py-2">Reason</th>
                          <th className="px-2 py-2">Created By</th>
                        </tr>
                      </thead>
                      <tbody>
                        {history.map((movement) => {
                          const reason = typeof movement.metadata?.reason === 'string'
                            ? movement.metadata.reason
                            : movement.reference_type === 'stock_transfer' && movement.reference_id
                              ? `Transfer #${movement.reference_id}`
                              : movement.reference_type ?? 'Manual'
                          const isPositive = movement.quantity_delta > 0

                          return (
                            <tr key={movement.id} className="border-t border-slate-100">
                              <td className="px-2 py-2 text-slate-600">{movement.created_at ? new Date(movement.created_at).toLocaleString() : '-'}</td>
                              <td className="px-2 py-2">
                                <span className={`inline-flex rounded-full px-2 py-0.5 font-medium ${isPositive ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'}`}>
                                  {movement.movement_type}
                                </span>
                              </td>
                              <td className={`px-2 py-2 font-medium ${isPositive ? 'text-emerald-700' : 'text-rose-700'}`}>
                                {movement.quantity_delta > 0 ? '+' : ''}{movement.quantity_delta}
                              </td>
                              <td className="px-2 py-2 text-slate-600">{reason}</td>
                              <td className="px-2 py-2 text-slate-600">{movement.actor_user_name ?? 'Unknown User'}</td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}

export default InventoryPage
