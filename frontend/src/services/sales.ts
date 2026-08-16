import type { ApiEnvelope } from '../types/auth'
import type { Sale, SaleFilter, CreateSaleInput } from '../types/sale'
import { api } from '../lib/api'

export const salesApi = {
  list: async (filter?: SaleFilter, token?: string): Promise<Sale[]> => {
    const q: string[] = []
    if (filter?.branch_id !== undefined) q.push(`branch_id=${filter.branch_id}`)
    const path = `/api/v1/sales${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Sale[]>>(path, token)
    return res.data ?? []
  },

  getById: async (id: number, token?: string): Promise<Sale> => {
    const res = await api.get<ApiEnvelope<Sale>>(`/api/v1/sales/${id}`, token)
    return res.data
  },

  create: async (payload: CreateSaleInput, token?: string): Promise<{ id: number; sale_number: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; sale_number: string }>>(`/api/v1/sales`, payload, token)
    return res.data
  },

  complete: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/sales/${id}/complete`, undefined, token)
    return res.data
  },

  cancel: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/sales/${id}/cancel`, undefined, token)
    return res.data
  },
}
