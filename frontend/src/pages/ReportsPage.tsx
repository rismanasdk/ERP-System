import { useCallback, useEffect, useMemo, useState } from 'react'
import { useBranch } from '../contexts/BranchContext'
import { useAuth } from '../hooks/useAuth'
import { branchesApi } from '../services/branches'
import { reportsApi } from '../services/reports'
import { readStoredAccessToken } from '../services/authSession'
import type { Branch } from '../types/auth'
import type { InventoryReport, PaymentReport, PurchaseReport, ReportsTab, SalesReport } from '../types/reporting'
import { ApiError } from '../lib/api'
import { formatDateShort } from '../utils/dateUtils'

const currency = (value: number) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value)

const getDefaultRange = () => {
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - 29)
  const pad = (value: number) => value.toString().padStart(2, '0')
  return {
    start_date: `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}`,
    end_date: `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}`,
    branch_id: '',
  }
}

type FilterState = {
  start_date: string
  end_date: string
  branch_id: string
  payment_method: string
}

function LoadingCard() {
  return (
    <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="h-4 w-24 rounded bg-slate-200" />
      <div className="mt-4 h-8 w-32 rounded bg-slate-200" />
      <div className="mt-3 h-4 w-40 rounded bg-slate-200" />
    </div>
  )
}

export function ReportsPage() {
  const { user } = useAuth()
  const canRead = user ? Boolean(user?.permissions?.includes('reports.read')) : true
  const { selectedBranch, isAllBranches } = useBranch()
  const token = readStoredAccessToken() ?? undefined
  const defaultRange = useMemo(() => getDefaultRange(), [])
  const [tab, setTab] = useState<ReportsTab>('sales')
  const [salesReport, setSalesReport] = useState<SalesReport | null>(null)
  const [purchaseReport, setPurchaseReport] = useState<PurchaseReport | null>(null)
  const [inventoryReport, setInventoryReport] = useState<InventoryReport | null>(null)
  const [paymentReport, setPaymentReport] = useState<PaymentReport | null>(null)
  const [branches, setBranches] = useState<Branch[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLoadingBranches, setIsLoadingBranches] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<FilterState>({
    start_date: defaultRange.start_date,
    end_date: defaultRange.end_date,
    branch_id: '',
    payment_method: '',
  })

  const loadBranches = useCallback(async () => {
    setIsLoadingBranches(true)
    try {
      const data = await branchesApi.list(true, token)
      setBranches(Array.isArray(data) ? data : [])
    } catch {
      setBranches([])
    } finally {
      setIsLoadingBranches(false)
    }
  }, [token])

  const loadSales = useCallback(async () => {
    if (!filter.start_date || !filter.end_date) {
      setError('Please select both start date and end date.')
      return
    }
    setIsLoading(true)
    setError(null)
    try {
      const payload = {
        start_date: filter.start_date,
        end_date: filter.end_date,
        branch_id: selectedBranch && selectedBranch.id > 0 && !isAllBranches ? selectedBranch.id : filter.branch_id ? Number(filter.branch_id) : undefined,
      }
      const data = await reportsApi.sales(payload, token)
      setSalesReport(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) setError('Session expired. Please sign in again.')
        else if (apiError.status === 403) setError('You do not have access to report data.')
        else setError(apiError.message || 'Unable to load sales report.')
        return
      }
      setError('Unable to load sales report.')
    } finally {
      setIsLoading(false)
    }
  }, [filter, isAllBranches, selectedBranch, token])

  const loadPurchases = useCallback(async () => {
    if (!filter.start_date || !filter.end_date) {
      setError('Please select both start date and end date.')
      return
    }
    setIsLoading(true)
    setError(null)
    try {
      const payload = {
        start_date: filter.start_date,
        end_date: filter.end_date,
        branch_id: selectedBranch && selectedBranch.id > 0 && !isAllBranches ? selectedBranch.id : filter.branch_id ? Number(filter.branch_id) : undefined,
      }
      const data = await reportsApi.purchases(payload, token)
      setPurchaseReport(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) setError('Session expired. Please sign in again.')
        else if (apiError.status === 403) setError('You do not have access to report data.')
        else setError(apiError.message || 'Unable to load purchases report.')
        return
      }
      setError('Unable to load purchases report.')
    } finally {
      setIsLoading(false)
    }
  }, [filter, isAllBranches, selectedBranch, token])

  const loadInventory = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const branchValue = selectedBranch && selectedBranch.id > 0 && !isAllBranches ? selectedBranch.id : filter.branch_id ? Number(filter.branch_id) : undefined
      const data = await reportsApi.inventory({ branch_id: branchValue }, token)
      setInventoryReport(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) setError('Session expired. Please sign in again.')
        else if (apiError.status === 403) setError('You do not have access to report data.')
        else setError(apiError.message || 'Unable to load inventory report.')
        return
      }
      setError('Unable to load inventory report.')
    } finally {
      setIsLoading(false)
    }
  }, [filter.branch_id, isAllBranches, selectedBranch, token])

  const loadPayments = useCallback(async () => {
    if (!filter.start_date || !filter.end_date) {
      setError('Please select both start date and end date.')
      return
    }
    setIsLoading(true)
    setError(null)
    try {
      const payload = {
        start_date: filter.start_date,
        end_date: filter.end_date,
        branch_id: selectedBranch && selectedBranch.id > 0 && !isAllBranches ? selectedBranch.id : filter.branch_id ? Number(filter.branch_id) : undefined,
        payment_method: filter.payment_method || undefined,
      }
      const data = await reportsApi.payments(payload, token)
      setPaymentReport(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) setError('Session expired. Please sign in again.')
        else if (apiError.status === 403) setError('You do not have access to report data.')
        else setError(apiError.message || 'Unable to load payment report.')
        return
      }
      setError('Unable to load payment report.')
    } finally {
      setIsLoading(false)
    }
  }, [filter, isAllBranches, selectedBranch, token])

  const fetchCurrentTab = useCallback(async () => {
    if (!canRead) return
    await loadBranches()
    if (tab === 'sales') await loadSales()
    else if (tab === 'purchases') await loadPurchases()
    else if (tab === 'inventory') await loadInventory()
    else await loadPayments()
  }, [canRead, loadBranches, loadInventory, loadPayments, loadPurchases, loadSales, tab])

  useEffect(() => {
    void (async () => {
      await fetchCurrentTab()
    })()
  }, [fetchCurrentTab])

  const tabConfig: Array<{ key: ReportsTab; label: string }> = [
    { key: 'sales', label: 'Sales' },
    { key: 'purchases', label: 'Purchasing' },
    { key: 'inventory', label: 'Inventory' },
    { key: 'payments', label: 'Payments' },
  ]

  const summaryCards = useMemo(() => {
    if (tab === 'sales') {
      return [
        { label: 'Total Sales', value: currency(salesReport?.total_sales ?? 0), description: 'Completed sales total' },
        { label: 'Total Transactions', value: (salesReport?.total_transactions ?? 0).toLocaleString('id-ID'), description: 'Completed transactions' },
        { label: 'Total Items Sold', value: (salesReport?.total_items_sold ?? 0).toLocaleString('id-ID'), description: 'Units sold' },
      ]
    }
    if (tab === 'purchases') {
      return [
        { label: 'Total Purchases', value: currency(purchaseReport?.total_purchases ?? 0), description: 'Completed purchase total' },
        { label: 'Transactions', value: (purchaseReport?.total_purchase_transactions ?? 0).toLocaleString('id-ID'), description: 'Purchase transactions' },
        { label: 'Items Purchased', value: (purchaseReport?.total_purchased_items ?? 0).toLocaleString('id-ID'), description: 'Units purchased' },
      ]
    }
    if (tab === 'inventory') {
      return [
        { label: 'Total Records', value: (inventoryReport?.total_inventory_records ?? 0).toLocaleString('id-ID'), description: 'Inventory rows' },
        { label: 'Total Quantity', value: (inventoryReport?.total_quantity ?? 0).toLocaleString('id-ID'), description: 'Units in stock' },
        { label: 'Low Stock', value: (inventoryReport?.low_stock_products?.length ?? 0).toLocaleString('id-ID'), description: 'Products below minimum' },
      ]
    }
    return [
      { label: 'Total Payments', value: (paymentReport?.total_payments ?? 0).toLocaleString('id-ID'), description: 'Payment records' },
      { label: 'Payment Amount', value: currency(paymentReport?.total_payment_amount ?? 0), description: 'Total collected' },
      { label: 'Methods', value: Object.keys(paymentReport?.method_breakdown ?? {}).length.toString(), description: 'Distinct methods' },
    ]
  }, [inventoryReport, paymentReport, purchaseReport, salesReport, tab])

  const renderTabContent = () => {
    if (tab === 'sales') {
      const rows = salesReport?.daily_summary ?? []
      return (
        <div className="space-y-4">
          {isLoading ? null : rows.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-600">No sales data found for this period.</div>
          ) : (
            <div className="overflow-auto">
              <table className="min-w-full table-auto">
                <thead>
                  <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                    <th className="px-3 py-3">Date</th>
                    <th className="px-3 py-3">Total Sales</th>
                    <th className="px-3 py-3">Transactions</th>
                    <th className="px-3 py-3">Items Sold</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.date} className="border-b border-slate-100 text-sm text-slate-700">
                      <td className="px-3 py-3">{formatDateShort(row.date)}</td>
                      <td className="px-3 py-3">{currency(row.total_sales)}</td>
                      <td className="px-3 py-3">{row.total_transactions}</td>
                      <td className="px-3 py-3">{row.total_items_sold}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )
    }
    if (tab === 'purchases') {
      const rows = purchaseReport?.daily_summary ?? []
      return (
        <div className="space-y-4">
          {isLoading ? null : rows.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-600">No purchase data found for this period.</div>
          ) : (
            <div className="overflow-auto">
              <table className="min-w-full table-auto">
                <thead>
                  <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                    <th className="px-3 py-3">Date</th>
                    <th className="px-3 py-3">Purchases</th>
                    <th className="px-3 py-3">Transactions</th>
                    <th className="px-3 py-3">Items</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.date} className="border-b border-slate-100 text-sm text-slate-700">
                      <td className="px-3 py-3">{formatDateShort(row.date)}</td>
                      <td className="px-3 py-3">{currency(row.total_purchases)}</td>
                      <td className="px-3 py-3">{row.total_transactions}</td>
                      <td className="px-3 py-3">{row.total_purchased_items}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )
    }
    if (tab === 'inventory') {
      const rows = inventoryReport?.rows ?? []
      return (
        <div className="space-y-4">
          {isLoading ? null : rows.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-600">No inventory rows found.</div>
          ) : (
            <div className="overflow-auto">
              <table className="min-w-full table-auto">
                <thead>
                  <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                    <th className="px-3 py-3">Product</th>
                    <th className="px-3 py-3">Branch</th>
                    <th className="px-3 py-3">Current Stock</th>
                    <th className="px-3 py-3">Min Stock</th>
                    <th className="px-3 py-3">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={`${row.branch_id}-${row.product_id}`} className="border-b border-slate-100 text-sm text-slate-700">
                      <td className="px-3 py-3">{row.product_name || row.product_id}</td>
                      <td className="px-3 py-3">{row.branch_name || row.branch_id}</td>
                      <td className="px-3 py-3">{row.current_stock}</td>
                      <td className="px-3 py-3">{row.minimum_stock}</td>
                      <td className="px-3 py-3">{row.stock_status || 'OK'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )
    }
    const rows = paymentReport?.rows ?? []
    return (
      <div className="space-y-4">
        {isLoading ? null : rows.length === 0 ? (
          <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-600">No payment data found for this period.</div>
        ) : (
          <div className="overflow-auto">
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Sale</th>
                  <th className="px-3 py-3">Method</th>
                  <th className="px-3 py-3">Amount</th>
                  <th className="px-3 py-3">Paid At</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id} className="border-b border-slate-100 text-sm text-slate-700">
                    <td className="px-3 py-3">{row.sale_number || row.id}</td>
                    <td className="px-3 py-3">{row.payment_method}</td>
                    <td className="px-3 py-3">{currency(row.amount)}</td>
                    <td className="px-3 py-3">{formatDateShort(row.paid_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    )
  }

  if (!canRead) {
    return <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">You do not have access to reports.</div>
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Reports</p>
        <h2 className="mt-1 text-2xl font-bold text-slate-900">Overview</h2>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex flex-wrap gap-2">
          {tabConfig.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => setTab(item.key)}
              className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                tab === item.key
                  ? 'bg-indigo-600 text-white'
                  : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="grid gap-4 md:grid-cols-4">
          <label className="block text-sm font-medium text-slate-700">
            Start Date
            <input
              type="date"
              value={filter.start_date}
              onChange={(e) => setFilter((current) => ({ ...current, start_date: e.target.value }))}
              className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            End Date
            <input
              type="date"
              value={filter.end_date}
              onChange={(e) => setFilter((current) => ({ ...current, end_date: e.target.value }))}
              className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            Branch
            <select
              value={filter.branch_id}
              onChange={(e) => setFilter((current) => ({ ...current, branch_id: e.target.value }))}
              disabled={isLoadingBranches}
              className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:cursor-not-allowed disabled:bg-slate-100"
            >
              <option value="">All branches</option>
              {branches.map((branch) => (
                <option key={branch.id} value={branch.id}>{branch.name}</option>
              ))}
            </select>
          </label>

          {tab === 'payments' ? (
            <label className="block text-sm font-medium text-slate-700">
              Method
              <select
                value={filter.payment_method}
                onChange={(e) => setFilter((current) => ({ ...current, payment_method: e.target.value }))}
                className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="">All methods</option>
                <option value="CASH">Cash</option>
                <option value="BANK_TRANSFER">Bank Transfer</option>
                <option value="QRIS">QRIS</option>
                <option value="OTHER">Other</option>
              </select>
            </label>
          ) : (
            <div className="flex items-end">
              <button
                type="button"
                onClick={() => {
                  if (tab === 'sales') void loadSales()
                  else if (tab === 'purchases') void loadPurchases()
                  else if (tab === 'inventory') void loadInventory()
                  else void loadPayments()
                }}
                className="w-full rounded-md bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
              >
                Apply
              </button>
            </div>
          )}

          {tab === 'payments' ? (
            <div className="flex items-end">
              <button
                type="button"
                onClick={() => void loadPayments()}
                className="w-full rounded-md bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
              >
                Apply
              </button>
            </div>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
          <div className="mt-3">
            <button
              type="button"
              onClick={() => {
                if (tab === 'sales') void loadSales()
                else if (tab === 'purchases') void loadPurchases()
                else if (tab === 'inventory') void loadInventory()
                else void loadPayments()
              }}
              className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors"
            >
              Retry
            </button>
          </div>
        </div>
      ) : null}

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <LoadingCard key={index} />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-3">
          {summaryCards.map((card) => (
            <div key={card.label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <p className="text-sm font-medium text-slate-500">{card.label}</p>
              <p className="mt-3 text-2xl font-bold text-slate-900">{card.value}</p>
              <p className="mt-2 text-sm text-slate-500">{card.description}</p>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="mb-4">
          <h3 className="text-lg font-semibold text-slate-900">
            {tab === 'sales' ? 'Daily Sales Summary' : tab === 'purchases' ? 'Daily Purchase Summary' : tab === 'inventory' ? 'Inventory Details' : 'Payment Details'}
          </h3>
        </div>
        {renderTabContent()}
      </div>
    </div>
  )
}

export default ReportsPage
