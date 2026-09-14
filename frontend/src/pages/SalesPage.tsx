import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import { useBranch } from '../contexts/BranchContext'
import type { Sale, CreateSaleInput, SaleFulfillment, SalesPayment, PaymentSummary } from '../types/sale'
import type { Branch } from '../types/auth'
import { salesApi } from '../services/sales'
import { inventoryApi } from '../services/inventory'
import { branchesApi } from '../services/branches'
import { readStoredAccessToken } from '../services/authSession'
import { CreateSaleForm } from '../components/sales/CreateSaleForm'
import { ApiError } from '../lib/api'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { formatDate, formatDateTime } from '../utils/dateUtils'
import {  ViewIcon, CloseIcon, CompleteIcon } from '../utils/iconsUtils'
import { customersApi } from '../services/customers'
import type { Customer } from '../types/customer'

const currency = (n: number) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)

export function SalesPage() {
  const { user } = useAuth()
  useBranch()
  const confirmDialog = useConfirm()
  const [items, setItems] = useState<Sale[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ branch_id: '' })
  const [creating, setCreating] = useState(false)
  const [viewingFor, setViewingFor] = useState<Sale | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [branchMap, setBranchMap] = useState<Record<number, Branch | null>>({})
  const [customerMap, setCustomerMap] = useState<Record<number, Customer | null>>({})
  const [fulfillments, setFulfillments] = useState<SaleFulfillment[]>([])
  const [fulfilling, setFulfilling] = useState(false)
  const [fulfillQuantities, setFulfillQuantities] = useState<Record<number, string>>({})
  const [fulfillError, setFulfillError] = useState<string | null>(null)
  const [currentStock, setCurrentStock] = useState<Record<number, number>>({})
  const [paymentSummary, setPaymentSummary] = useState<PaymentSummary | null>(null)
  const [payments, setPayments] = useState<SalesPayment[]>([])
  const [paymentOpen, setPaymentOpen] = useState(false)
  const [paymentForm, setPaymentForm] = useState({ amount: '', payment_method: 'CASH', reference_number: '', notes: '' })
  const [paymentError, setPaymentError] = useState<string | null>(null)

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

  const canRead = user ? Boolean(user?.permissions?.includes('sales.read')) : true

  const load = useCallback(async () => {
    if (!canRead) {
      setIsLoading(false)
      setError(null)
      return
    }

    setIsLoading(true)
    setError(null)
    try {
      const list = await salesApi.list({ branch_id: filter.branch_id ? Number(filter.branch_id) : undefined }, token)
      const data = (list ?? []) as Sale[]
      setItems(data)
      await fetchMissingBranches([...new Set(data.map((d) => d.branch_id))])
      const customerEntries = await Promise.all([...new Set(data.map((d) => d.customer_id).filter(Boolean))].map(async (id) => { try { return [id, await customersApi.getById(id, token)] as const } catch { return [id, null] as const } }))
      setCustomerMap((current) => ({ ...current, ...Object.fromEntries(customerEntries) }))
    } catch (err) {
      const e = err as ApiError
      if (e.status === 403) {
        setError('You do not have access to sales.')
      } else {
        setError(e.message)
      }
    } finally {
      setIsLoading(false)
    }
  }, [canRead, token, filter.branch_id, fetchMissingBranches])

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
    const [detail, history, summary, paymentHistory] = await Promise.all([salesApi.getById(row.id, token), salesApi.listFulfillments(row.id, token), salesApi.paymentSummary(row.id, token), salesApi.listPayments(row.id, token)])
    setViewingFor(detail)
    setFulfillments(Array.isArray(history) ? history : [])
    setPaymentSummary(summary)
    setPayments(Array.isArray(paymentHistory) ? paymentHistory : [])
  }, [token])

  const onPayment = useCallback(async () => {
    if (!viewingFor || !paymentSummary) return
    const amount = Number(paymentForm.amount)
    if (!Number.isFinite(amount) || amount <= 0 || amount > paymentSummary.remaining_amount) { setPaymentError('Enter an amount within the remaining balance.'); return }
    setSubmitting(true); setPaymentError(null)
    try { await salesApi.createPayment(viewingFor.id, { amount, payment_method: paymentForm.payment_method, reference_number: paymentForm.reference_number || undefined, notes: paymentForm.notes || undefined }, token); setPaymentOpen(false); setPaymentForm({ amount: '', payment_method: 'CASH', reference_number: '', notes: '' }); const [summary, history] = await Promise.all([salesApi.paymentSummary(viewingFor.id, token), salesApi.listPayments(viewingFor.id, token)]); setPaymentSummary(summary); setPayments(history) } catch (err) { setPaymentError(err instanceof ApiError ? err.message : 'Unable to create payment.') } finally { setSubmitting(false) }
  }, [paymentForm, paymentSummary, token, viewingFor])

  const onConfirm = useCallback(async (id: number) => { setSubmitting(true); try { await salesApi.confirm(id, token); await load() } catch (err) { setError(err instanceof ApiError ? err.message : 'Unable to confirm sale.') } finally { setSubmitting(false) } }, [load, token])

  const onFulfill = useCallback(async () => {
    if (!viewingFor?.items) return
    const payload = viewingFor.items.map((item) => ({ sales_order_item_id: item.id, quantity_fulfilled: Number(fulfillQuantities[item.id] || 0) })).filter((item) => item.quantity_fulfilled > 0)
    const invalid = !payload.length || payload.some((item) => { const original = viewingFor.items?.find((candidate) => candidate.id === item.sales_order_item_id); return !original || item.quantity_fulfilled > original.quantity - original.fulfilled_quantity })
    if (invalid) { setFulfillError('Enter a valid quantity within the remaining amount.'); return }
    setSubmitting(true); setFulfillError(null)
    try { await salesApi.fulfill(viewingFor.id, { items: payload }, token); setFulfilling(false); await load(); const [detail, history] = await Promise.all([salesApi.getById(viewingFor.id, token), salesApi.listFulfillments(viewingFor.id, token)]); setViewingFor(detail); setFulfillments(history) } catch (err) { setFulfillError(err instanceof ApiError ? err.message : 'Unable to fulfill sale.') } finally { setSubmitting(false) }
  }, [fulfillQuantities, load, token, viewingFor])

  const openFulfill = useCallback(async (sale: Sale) => {
    await onView(sale)
    setFulfilling(true)
    try {
      const stock = await inventoryApi.list({ branch_id: sale.branch_id })
      setCurrentStock(Object.fromEntries(stock.map((item) => [item.product_id, item.quantity])))
    } catch { setCurrentStock({}) }
  }, [onView])

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

  const canCreate = user?.permissions?.includes('sales.create')
  const canComplete = user?.permissions?.includes('sales.complete')
  const canConfirm = user?.permissions?.includes('sales.confirm')
  const canFulfill = user?.permissions?.includes('sales.fulfill')
  const canCancel = user?.permissions?.includes('sales.cancel')
  const canPaymentRead = user?.permissions?.includes('sales.payment.read')
  const canPaymentCreate = user?.permissions?.includes('sales.payment.create')

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
                  <th className="px-3 py-3">Customer</th>
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
                    <td className="px-3 py-3 text-sm text-slate-700">{customerMap[p.customer_id]?.name ?? `#${p.customer_id}`}</td>
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
    <button onClick={() => void onView(p)} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
      <ViewIcon className="h-3.5 w-3.5" />
      View
    </button>
    {p.status === 'DRAFT' && canComplete ? (
      <button onClick={() => void onComplete(p.id)} className="inline-flex items-center gap-1.5 rounded-md border border-green-300 bg-white px-2.5 py-1.5 text-xs font-medium text-green-700 hover:bg-green-50 transition-colors">
        <CompleteIcon className="h-3.5 w-3.5" />
        Complete
      </button>
    ) : null}
    {p.status === 'DRAFT' && canConfirm ? <button onClick={() => void onConfirm(p.id)} className="rounded-md border border-indigo-300 bg-white px-2.5 py-1.5 text-xs font-medium text-indigo-700">Confirm</button> : null}
    {(p.status === 'CONFIRMED' || p.status === 'PARTIALLY_FULFILLED') && canFulfill ? <button onClick={() => { void openFulfill(p); setFulfillError(null) }} className="rounded-md border border-indigo-300 bg-white px-2.5 py-1.5 text-xs font-medium text-indigo-700">Fulfill</button> : null}
    {p.status !== 'CANCELLED' && canCancel ? (
      <button onClick={() => void onCancel(p.id)} className="inline-flex items-center gap-1.5 rounded-md border border-red-300 bg-white px-2.5 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 transition-colors">
        <CloseIcon className="h-3.5 w-3.5" />
        Cancel
      </button>
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
                <div className="text-sm text-slate-500">Customer</div>
                <div className="text-sm font-medium">{customerMap[viewingFor.customer_id]?.name ?? `#${viewingFor.customer_id}`}</div>
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

              <div><div className="text-sm text-slate-500">Items</div><table className="mt-2 min-w-full text-left text-xs"><thead><tr><th className="px-2 py-2">Product</th><th className="px-2 py-2">Ordered</th><th className="px-2 py-2">Fulfilled</th><th className="px-2 py-2">Remaining</th></tr></thead><tbody>{(viewingFor.items ?? []).map((item) => <tr key={item.id} className="border-t border-slate-100"><td className="px-2 py-2">{item.product_id}</td><td className="px-2 py-2">{item.quantity}</td><td className="px-2 py-2">{item.fulfilled_quantity}</td><td className="px-2 py-2">{item.quantity - item.fulfilled_quantity}</td></tr>)}</tbody></table></div>
              {viewingFor.status === 'DRAFT' && canConfirm ? <button onClick={() => void onConfirm(viewingFor.id)} className="rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">Confirm Order</button> : null}
              {(viewingFor.status === 'CONFIRMED' || viewingFor.status === 'PARTIALLY_FULFILLED') && canFulfill ? <button onClick={() => void openFulfill(viewingFor)} className="rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">Fulfill Order</button> : null}
              {fulfillments.length > 0 ? <div><div className="text-sm text-slate-500">Fulfillment History</div>{fulfillments.map((fulfillment) => <div key={fulfillment.id} className="mt-1 text-sm">Fulfillment #{fulfillment.id}: {fulfillment.items.reduce((sum, item) => sum + item.quantity_fulfilled, 0)} units, {fulfillment.fulfilled_by_name || 'Unknown User'}</div>)}</div> : null}
              {canPaymentRead && paymentSummary ? <div className="border-t border-slate-200 pt-4"><div className="mb-2 text-sm font-semibold text-slate-700">Payment Summary</div><div className="grid grid-cols-2 gap-2 text-sm"><span>Total</span><span className="text-right">{currency(paymentSummary.order_total)}</span><span>Paid</span><span className="text-right">{currency(paymentSummary.paid_amount)}</span><span>Remaining</span><span className="text-right">{currency(paymentSummary.remaining_amount)}</span><span>Status</span><span className="text-right font-medium">{paymentSummary.payment_status}</span></div>{canPaymentCreate && paymentSummary.remaining_amount > 0 && viewingFor.status !== 'DRAFT' && viewingFor.status !== 'CANCELLED' ? <button onClick={() => { setPaymentOpen(true); setPaymentError(null) }} className="mt-3 rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">Add Payment</button> : null}<div className="mt-3"><div className="text-sm text-slate-500">Payment History</div>{payments.map((payment) => <div key={payment.id} className="mt-1 flex justify-between text-sm"><span>{payment.paid_at ? formatDate(payment.paid_at) : '-'} · {payment.payment_method} · {payment.reference_number || '-'}</span><span>{currency(payment.amount)} · {payment.created_by_name || 'Unknown User'}</span></div>)}</div></div> : null}
            </div>
          </DialogContent>
        </Dialog>
      )}
      {fulfilling && viewingFor ? <Dialog open={fulfilling} onOpenChange={setFulfilling}><DialogContent><h3 className="text-lg font-semibold text-slate-900">Fulfill Order</h3><div className="mt-4 space-y-3">{(viewingFor.items ?? []).map((item) => <label key={item.id} className="block text-sm text-slate-700">Product {item.product_id} (remaining {item.quantity - item.fulfilled_quantity}, current stock {currentStock[item.product_id] ?? 0})<input type="number" min="0" max={item.quantity - item.fulfilled_quantity} value={fulfillQuantities[item.id] ?? ''} onChange={(e) => setFulfillQuantities((current) => ({ ...current, [item.id]: e.target.value }))} className="mt-1 block w-full rounded-md border-slate-200" /></label>)}{fulfillError ? <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{fulfillError}</div> : null}<div className="flex justify-end gap-2"><button onClick={() => setFulfilling(false)} className="rounded-md border border-slate-200 px-3 py-2 text-sm">Cancel</button><button onClick={() => void onFulfill()} disabled={submitting} className="rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">Fulfill Order</button></div></div></DialogContent></Dialog> : null}
      {paymentOpen && viewingFor && paymentSummary ? <Dialog open={paymentOpen} onOpenChange={setPaymentOpen}><DialogContent><h3 className="text-lg font-semibold text-slate-900">Add Payment</h3><div className="mt-4 space-y-3"><div className="rounded-md bg-slate-50 p-3 text-sm text-slate-600">Remaining: <span className="font-semibold text-slate-900">{currency(paymentSummary.remaining_amount)}</span></div><label className="block text-sm text-slate-700">Amount<input type="number" min="0.01" max={paymentSummary.remaining_amount} value={paymentForm.amount} onChange={(e) => setPaymentForm((form) => ({ ...form, amount: e.target.value }))} className="mt-1 block w-full rounded-md border-slate-200" /></label><label className="block text-sm text-slate-700">Payment Method<select value={paymentForm.payment_method} onChange={(e) => setPaymentForm((form) => ({ ...form, payment_method: e.target.value }))} className="mt-1 block w-full rounded-md border-slate-200"><option value="CASH">CASH</option><option value="BANK_TRANSFER">BANK_TRANSFER</option><option value="OTHER">OTHER</option><option value="QRIS">QRIS</option></select></label><label className="block text-sm text-slate-700">Reference Number<input value={paymentForm.reference_number} onChange={(e) => setPaymentForm((form) => ({ ...form, reference_number: e.target.value }))} className="mt-1 block w-full rounded-md border-slate-200" /></label><label className="block text-sm text-slate-700">Notes<textarea value={paymentForm.notes} onChange={(e) => setPaymentForm((form) => ({ ...form, notes: e.target.value }))} className="mt-1 block w-full rounded-md border-slate-200" rows={2} /></label>{paymentError ? <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{paymentError}</div> : null}<div className="flex justify-end gap-2"><button onClick={() => setPaymentOpen(false)} className="rounded-md border border-slate-200 px-3 py-2 text-sm">Cancel</button><button onClick={() => void onPayment()} disabled={submitting} className="rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">Save Payment</button></div></div></DialogContent></Dialog> : null}
    </div>
  )
}

export default SalesPage