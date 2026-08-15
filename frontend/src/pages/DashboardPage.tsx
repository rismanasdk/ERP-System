import { useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import { api, ApiError } from '../lib/api'
import type { DashboardSummary } from '../types/dashboard'
import { readStoredAccessToken } from '../services/authSession'

function formatCurrency(value: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value)
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('id-ID').format(value)
}

function StatCard({
  label,
  value,
  subtext,
}: {
  label: string
  value: string
  subtext?: string
}) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <p className="text-sm font-medium text-slate-500">{label}</p>
      <p className="mt-3 text-2xl font-bold text-slate-900">{value}</p>
      {subtext ? <p className="mt-2 text-sm text-slate-500">{subtext}</p> : null}
    </div>
  )
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

function ErrorState({ message, retry }: { message: string; retry?: () => void }) {
  return (
    <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
      <h3 className="text-lg font-semibold">Dashboard unavailable</h3>
      <p className="mt-2 text-sm">{message}</p>
      {retry ? (
        <button
          type="button"
          onClick={retry}
          className="mt-4 rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-500"
        >
          Retry
        </button>
      ) : null}
    </div>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadDashboard = async () => {
    setIsLoading(true)
    setError(null)

    try {
      const token = readStoredAccessToken() ?? undefined
      const data = await api.getDashboardSummary<DashboardSummary>(token)
      setSummary(data)
    } catch (err) {
      const apiError = err as ApiError
      if (apiError instanceof ApiError) {
        if (apiError.status === 401) {
          setError('Session expired. Please sign in again.')
          return
        }
        if (apiError.status === 403) {
          setError('You do not have access to this dashboard.')
          return
        }
        setError(apiError.message || 'Unable to load dashboard data.')
        return
      }
      setError('Unable to load dashboard data.')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    let isActive = true

    const fetchData = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const token = readStoredAccessToken() ?? undefined
        const data = await api.getDashboardSummary<DashboardSummary>(token)
        if (isActive) {
          setSummary(data)
        }
      } catch (err) {
        if (!isActive) {
          return
        }

        const apiError = err as ApiError
        if (apiError instanceof ApiError) {
          if (apiError.status === 401) {
            setError('Session expired. Please sign in again.')
            return
          }
          if (apiError.status === 403) {
            setError('You do not have access to this dashboard.')
            return
          }
          setError(apiError.message || 'Unable to load dashboard data.')
          return
        }
        setError('Unable to load dashboard data.')
      } finally {
        if (isActive) {
          setIsLoading(false)
        }
      }
    }

    void fetchData()

    return () => {
      isActive = false
    }
  }, [])

  const metrics = useMemo(() => {
    const sales = summary?.sales ?? null
    const purchases = summary?.purchases ?? null
    const masterData = summary?.master_data ?? null
    const inventory = summary?.inventory ?? null

    return {
      salesToday: sales ? formatCurrency(sales.today_amount) : 'Rp 0',
      salesMonth: sales ? formatCurrency(sales.month_amount) : 'Rp 0',
      salesTransactions: sales ? formatNumber(sales.today_transactions) : '0',
      purchasesToday: purchases ? formatCurrency(purchases.today_amount) : 'Rp 0',
      purchasesMonth: purchases ? formatCurrency(purchases.month_amount) : 'Rp 0',
      purchasesTransactions: purchases ? formatNumber(purchases.today_transactions) : '0',
      productCount: masterData ? formatNumber(masterData.products) : '0',
      customerCount: masterData ? formatNumber(masterData.customers) : '0',
      supplierCount: masterData ? formatNumber(masterData.suppliers) : '0',
      inventoryCount: inventory ? formatNumber(inventory.total_items) : '0',
    }
  }, [summary])

  const branchLabel = user?.roles?.includes('SUPER_ADMIN') ? 'All active branches' : 'Your assigned branch'

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Overview</p>
        <h2 className="mt-3 text-3xl font-bold text-slate-900">Dashboard</h2>
        <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <span>{user?.name ?? user?.email ?? 'User'}</span>
          <span className="text-slate-300">•</span>
          <span>{branchLabel}</span>
        </div>
      </div>

      {error ? (
        <ErrorState message={error} retry={() => void loadDashboard()} />
      ) : null}

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <LoadingCard key={index} />
          ))}
        </div>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <StatCard label="Sales Today" value={metrics.salesToday} subtext={`${metrics.salesTransactions} transactions`} />
            <StatCard label="Sales Month" value={metrics.salesMonth} subtext="Current month" />
            <StatCard label="Purchases Today" value={metrics.purchasesToday} subtext={`${metrics.purchasesTransactions} transactions`} />
            <StatCard label="Inventory" value={metrics.inventoryCount} subtext="Total stock items" />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <p className="text-sm font-medium uppercase tracking-[0.2em] text-slate-500">Sales</p>
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <span className="text-slate-600">Today</span>
                  <span className="font-semibold text-slate-900">{metrics.salesToday}</span>
                </div>
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <span className="text-slate-600">This month</span>
                  <span className="font-semibold text-slate-900">{metrics.salesMonth}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-600">Transactions</span>
                  <span className="font-semibold text-slate-900">{metrics.salesTransactions}</span>
                </div>
              </div>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <p className="text-sm font-medium uppercase tracking-[0.2em] text-slate-500">Purchases</p>
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <span className="text-slate-600">Today</span>
                  <span className="font-semibold text-slate-900">{metrics.purchasesToday}</span>
                </div>
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <span className="text-slate-600">This month</span>
                  <span className="font-semibold text-slate-900">{metrics.purchasesMonth}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-600">Transactions</span>
                  <span className="font-semibold text-slate-900">{metrics.purchasesTransactions}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            <StatCard label="Active Products" value={metrics.productCount} />
            <StatCard label="Active Customers" value={metrics.customerCount} />
            <StatCard label="Active Suppliers" value={metrics.supplierCount} />
          </div>
        </>
      )}
    </div>
  )
}
