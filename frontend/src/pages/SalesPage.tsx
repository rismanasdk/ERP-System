import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Sale, CreateSaleInput } from '../types/sale'
import { salesApi } from '../services/sales'
import { readStoredAccessToken } from '../services/authSession'
import { CreateSaleForm } from '../components/sales/CreateSaleForm'
import { ApiError } from '../lib/api'

export function SalesPage() {
  const { user } = useAuth()
  const [items, setItems] = useState<Sale[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await salesApi.list(undefined, token)
      setItems(res)
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
  }, [token])

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
    if (!confirm('Selesaikan penjualan ini? This will decrease inventory.')) return
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
  }, [load, token])

  const onCancel = useCallback(async (id: number) => {
    if (!confirm('Batalkan penjualan ini? This may restore inventory if already completed.')) return
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
  }, [load, token])

  const canCreate = user?.permissions?.includes('sales.create')
  const canComplete = user?.permissions?.includes('sales.complete')
  const canCancel = user?.permissions?.includes('sales.cancel')

  const rows = useMemo(() => items, [items])

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
            <button onClick={() => setCreating(true)} className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500">Buat Penjualan</button>
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
                <tr className="text-left text-sm text-slate-500">
                  <th className="px-3 py-2">Number</th>
                  <th className="px-3 py-2">Status</th>
                  <th className="px-3 py-2">Total</th>
                  <th className="px-3 py-2">Created By</th>
                  <th className="px-3 py-2">Created</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((p) => (
                  <tr key={p.id} className="border-t">
                    <td className="px-3 py-2 align-top">{p.sale_number}</td>
                    <td className="px-3 py-2 align-top">{p.status}</td>
                    <td className="px-3 py-2 align-top">{new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(p.total_amount)}</td>
                    <td className="px-3 py-2 align-top">{p.created_by}</td>
                    <td className="px-3 py-2 align-top">{p.created_at ?? '-'}</td>
                    <td className="px-3 py-2 align-top">
                      <div className="flex gap-2">
                        {p.status === 'DRAFT' && canComplete ? <button onClick={() => void onComplete(p.id)} className="rounded-md border border-green-300 bg-white px-2 py-1 text-sm text-green-700">Selesaikan</button> : null}
                        {p.status !== 'CANCELLED' && canCancel ? <button onClick={() => void onCancel(p.id)} className="rounded-md border border-red-300 bg-white px-2 py-1 text-sm text-red-700">Batalkan</button> : null}
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
          <h3 className="text-lg font-semibold">Buat Penjualan</h3>
          <div className="mt-4">
            <CreateSaleForm submitting={submitting} onSubmit={onCreate} onCancel={() => setCreating(false)} />
          </div>
        </div>
      )}
    </div>
  )
}

export default SalesPage
