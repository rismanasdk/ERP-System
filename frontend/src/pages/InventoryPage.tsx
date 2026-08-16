import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Inventory } from '../types/inventory'
import { inventoryApi } from '../services/inventory'
import { readStoredAccessToken } from '../services/authSession'
import { InventoryAdjustForm } from '../components/inventory/InventoryAdjustForm'
import { InventoryCreateForm } from '../components/inventory/InventoryCreateForm'
import { ApiError } from '../lib/api'

export function InventoryPage() {
  const { user } = useAuth()
  const [items, setItems] = useState<Inventory[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '', product_id: '' })
  const [adjustingFor, setAdjustingFor] = useState<Inventory | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const f: { branch_id?: number; product_id?: number } = {}
      if (filter.branch_id) f.branch_id = Number(filter.branch_id)
      if (filter.product_id) f.product_id = Number(filter.product_id)
      const res = await inventoryApi.list(f, token)
      setItems(res)
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
  }, [filter, token])

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

  const rows = useMemo(() => items, [items])

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
            <button onClick={() => setCreating(true)} className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500">Create</button>
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
          <input value={filter.product_id} onChange={(e) => setFilter((s) => ({ ...s, product_id: e.target.value }))} placeholder="Filter by product id" className="rounded-md border border-slate-200 px-3 py-2" />
          <input value={filter.branch_id} onChange={(e) => setFilter((s) => ({ ...s, branch_id: e.target.value }))} placeholder="Filter by branch id" className="rounded-md border border-slate-200 px-3 py-2" />
          <button onClick={() => void load()} className="rounded-md border border-slate-300 bg-white px-3 py-2">Apply</button>
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
                <tr className="text-left text-sm text-slate-500">
                  <th className="px-3 py-2">Product ID</th>
                  <th className="px-3 py-2">Branch ID</th>
                  <th className="px-3 py-2">Quantity</th>
                  <th className="px-3 py-2">Updated</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((i) => (
                  <tr key={i.id} className="border-t">
                    <td className="px-3 py-2 align-top">{i.product_id}</td>
                    <td className="px-3 py-2 align-top">{i.branch_id}</td>
                    <td className="px-3 py-2 align-top">{i.quantity}</td>
                    <td className="px-3 py-2 align-top">{i.updated_at ?? i.created_at ?? '-'}</td>
                    <td className="px-3 py-2 align-top">
                      <div className="flex gap-2">
                        {canAdjust ? <button onClick={() => setAdjustingFor(i)} className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm">Adjust</button> : null}
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
          <h3 className="text-lg font-semibold">Create inventory</h3>
          <div className="mt-4">
            <InventoryCreateForm
              key="create-inventory"
              submitting={submitting}
              onSubmit={onCreate}
              onCancel={() => setCreating(false)}
            />
          </div>
        </div>
      )}

      {adjustingFor && (
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold">Adjust inventory for product {adjustingFor.product_id} (branch {adjustingFor.branch_id})</h3>
          <div className="mt-4">
            <InventoryAdjustForm
              key={`adjust-${adjustingFor.id}`}
              initial={{}}
              submitting={submitting}
              onSubmit={onAdjust}
              onCancel={() => setAdjustingFor(null)}
            />
          </div>
        </div>
      )}
    </div>
  )
}

export default InventoryPage
