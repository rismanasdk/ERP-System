import type { ApiEnvelope } from '../types/auth'
import type { InventoryReport, PaymentReport, PurchaseReport, SalesReport, SalesReportFilter } from '../types/reporting'
import { api } from '../lib/api'

const getSalesReport = async (filter: SalesReportFilter = {}, token?: string): Promise<SalesReport> => {
  const params = new URLSearchParams()
  if (filter.start_date) params.set('start_date', filter.start_date)
  if (filter.end_date) params.set('end_date', filter.end_date)
  if (filter.branch_id !== undefined && filter.branch_id !== null && filter.branch_id > 0) {
    params.set('branch_id', String(filter.branch_id))
  }
  const path = `/api/v1/reports/sales${params.toString() ? `?${params}` : ''}`
  const res = await api.get<ApiEnvelope<SalesReport>>(path, token)
  return res.data
}

export const reportsApi = {
  sales: getSalesReport,
  purchases: async (filter: SalesReportFilter = {}, token?: string): Promise<PurchaseReport> => {
    const params = new URLSearchParams()
    if (filter.start_date) params.set('start_date', filter.start_date)
    if (filter.end_date) params.set('end_date', filter.end_date)
    if (filter.branch_id !== undefined && filter.branch_id !== null && filter.branch_id > 0) {
      params.set('branch_id', String(filter.branch_id))
    }
    const path = `/api/v1/reports/purchases${params.toString() ? `?${params}` : ''}`
    const res = await api.get<ApiEnvelope<PurchaseReport>>(path, token)
    return res.data
  },
  inventory: async (filter: { branch_id?: number; product_id?: number } = {}, token?: string): Promise<InventoryReport> => {
    const params = new URLSearchParams()
    if (filter.branch_id !== undefined && filter.branch_id !== null && filter.branch_id > 0) {
      params.set('branch_id', String(filter.branch_id))
    }
    if (filter.product_id !== undefined && filter.product_id !== null && filter.product_id > 0) {
      params.set('product_id', String(filter.product_id))
    }
    const path = `/api/v1/reports/inventory${params.toString() ? `?${params}` : ''}`
    const res = await api.get<ApiEnvelope<InventoryReport>>(path, token)
    return res.data
  },
  payments: async (filter: SalesReportFilter & { payment_method?: string } = {}, token?: string): Promise<PaymentReport> => {
    const params = new URLSearchParams()
    if (filter.start_date) params.set('start_date', filter.start_date)
    if (filter.end_date) params.set('end_date', filter.end_date)
    if (filter.branch_id !== undefined && filter.branch_id !== null && filter.branch_id > 0) {
      params.set('branch_id', String(filter.branch_id))
    }
    if (filter.payment_method) params.set('payment_method', filter.payment_method)
    const path = `/api/v1/reports/payments${params.toString() ? `?${params}` : ''}`
    const res = await api.get<ApiEnvelope<PaymentReport>>(path, token)
    return res.data
  },
}

export const salesReportsApi = {
  get: getSalesReport,
  sales: getSalesReport,
  purchases: reportsApi.purchases,
  inventory: reportsApi.inventory,
  payments: reportsApi.payments,
}
