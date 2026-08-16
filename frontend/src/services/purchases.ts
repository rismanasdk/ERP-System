import type { ApiEnvelope } from '../types/auth'
import type { Purchase, PurchaseFilter, CreatePurchaseInput } from '../types/purchase'
import { api } from '../lib/api'

export const purchasesApi = {
  list: async (filter?: PurchaseFilter, token?: string): Promise<Purchase[]> => {
    const q: string[] = []
    if (filter?.branch_id !== undefined) q.push(`branch_id=${filter.branch_id}`)
    if (filter?.supplier_id !== undefined) q.push(`supplier_id=${filter.supplier_id}`)
    if (filter?.status !== undefined) q.push(`status=${encodeURIComponent(filter.status)}`)
    const path = `/api/v1/purchases${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Purchase[]>>(path, token)
    return res.data ?? []
  },


  
  getById: async (id: number, token?: string): Promise<Purchase> => {
    const res = await api.get<ApiEnvelope<Purchase>>(`/api/v1/purchases/${id}`, token)
    return res.data
  },

  create: async (payload: CreatePurchaseInput, token?: string): Promise<{ id: number; purchase_number: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; purchase_number: string }>>(`/api/v1/purchases`, payload, token)
    return res.data
  },

  complete: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/purchases/${id}/complete`, undefined, token)
    return res.data
  },

  cancel: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/purchases/${id}/cancel`, undefined, token)
    return res.data
  },
}
