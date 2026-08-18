import { useCallback, useEffect, useMemo, useState } from 'react'
import { useBranch } from '../contexts/BranchContext'
import { branchesApi } from '../services/branches'
import { salesReportsApi } from '../services/reports'
import { readStoredAccessToken } from '../services/authSession'
import type { Branch } from '../types/auth'
import type { SalesReport } from '../types/reporting'
import { ApiError } from '../lib/api'
import { formatDateShort } from '../utils/dateUtils'

const currency = (value: number) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value)

const toDateInputValue = (date: Date) => {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 10)
}

const getDefaultRange = () => {
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - 29)

  return {
    start_date: toDateInputValue(start),
    end_date: toDateInputValue(end),
    branch_id: '',
  }
}

type FilterState = {
  start_date: string
  end_date: string
  branch_id: string
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

export function SalesReportPage() {
  const { selectedBranch, isAllBranches } = useBranch()
  const token = readStoredAccessToken() ?? undefined
  const defaultRange = useMemo(() => getDefaultRange(), [])
  const [report, setReport] = useState<SalesReport | null>(null)
  const [branches, setBranches] = useState<Branch[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLoadingBranches, setIsLoadingBranches] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<FilterState>({
    start_date: defaultRange.start_date,
    end_date: defaultRange.end_date,
    branch_id: '',
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

  const loadReport = useCallback(async (nextFilter: FilterState) => {
    if (!nextFilter.start_date || !nextFilter.end_date) {
      setError('Please select both start date and end date.')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const payload = {
        start_date: nextFilter.start_date,
        end_date: nextFilter.end_date,
        branch_id: selectedBranch && selectedBranch.id > 0 && !isAllBranches
          ? selectedBranch.id
          : nextFilter.branch_id ? Number(nextFilter.branch_id) : undefined,
      }
      const data = await salesReportsApi.get(payload, token)
      setReport(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) {
          setError('Session expired. Please sign in again.')
          return
        }
        if (apiError.status === 403) {
          setError('You do not have access to sales reports.')
          return
        }
        setError(apiError.message || 'Unable to load sales report.')
        return
      }
      setError('Unable to load sales report.')
    } finally {
      setIsLoading(false)
    }
  }, [isAllBranches, selectedBranch, token])

  useEffect(() => {
    void (async () => {
      await loadBranches()
      await loadReport(defaultRange)
    })()
  }, [defaultRange, loadBranches, loadReport])

  const summaryCards = useMemo(() => [
    {
      label: 'Total Sales',
      value: currency(report?.total_sales ?? 0),
      description: 'Completed sales total',
    },
    {
      label: 'Total Transactions',
      value: (report?.total_transactions ?? 0).toLocaleString('id-ID'),
      description: 'Completed transactions',
    },
    {
      label: 'Total Items Sold',
      value: (report?.total_items_sold ?? 0).toLocaleString('id-ID'),
      description: 'Units sold',
    },
  ], [report])

  const dailyRows = report?.daily_summary ?? []

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Reports</p>
        <h2 className="mt-1 text-2xl font-bold text-slate-900">Sales Report</h2>
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

          <div className="flex items-end">
            <button
              type="button"
              onClick={() => void loadReport(filter)}
              className="w-full rounded-md bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
            >
              Apply
            </button>
          </div>
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
          <div className="mt-3">
            <button
              type="button"
              onClick={() => void loadReport(filter)}
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
          <h3 className="text-lg font-semibold text-slate-900">Daily Sales Summary</h3>
        </div>

        {isLoading ? null : dailyRows.length === 0 ? (
          <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-600">
            No sales data found for this period.
          </div>
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
              <tbody className="divide-y divide-slate-100">
                {dailyRows.map((row) => (
                  <tr key={row.date} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm text-slate-700">{formatDateShort(row.date)}</td>
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{currency(row.total_sales)}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{row.total_transactions}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{row.total_items_sold}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

export default SalesReportPage
