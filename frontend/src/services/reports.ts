import type { ApiEnvelope } from '../types/auth'
import type { SalesReport, SalesReportFilter } from '../types/reporting'
import { api } from '../lib/api'

export const salesReportsApi = {
  get: async (filter: SalesReportFilter = {}, token?: string): Promise<SalesReport> => {
    const params = new URLSearchParams()

    if (filter.start_date) params.set('start_date', filter.start_date)
    if (filter.end_date) params.set('end_date', filter.end_date)
    if (filter.branch_id !== undefined && filter.branch_id !== null && filter.branch_id > 0) {
      params.set('branch_id', String(filter.branch_id))
    }

    const query = params.toString()
    const path = `/api/v1/reports/sales${query ? `?${query}` : ''}`
    const res = await api.get<ApiEnvelope<SalesReport>>(path, token)
    return res.data
  },
}
