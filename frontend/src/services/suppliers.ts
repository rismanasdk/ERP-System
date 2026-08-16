import type { ApiEnvelope } from '../types/auth'
import type { Supplier, SupplierFilter } from '../types/supplier'
import { api } from '../lib/api'

export const suppliersApi = {
  list: async (filter?: SupplierFilter, token?: string): Promise<Supplier[]> => {
    const q = [] as string[]
    if (filter?.search) q.push(`search=${encodeURIComponent(filter.search)}`)
    if (filter?.active !== undefined) q.push(`active=${filter.active}`)
    const path = `/api/v1/suppliers${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Supplier[]>>(path, token)
    return res.data
  },

  getById: async (id: number, token?: string): Promise<Supplier> => {
    const res = await api.get<ApiEnvelope<Supplier>>(`/api/v1/suppliers/${id}`, token)
    return res.data
  },

  create: async (payload: Partial<Supplier>, token?: string): Promise<Supplier> => {
    const res = await api.post<ApiEnvelope<Supplier>>(`/api/v1/suppliers`, payload, token)
    return res.data
  },

  update: async (id: number, payload: Partial<Supplier>, token?: string): Promise<Supplier> => {
    const res = await api.put<ApiEnvelope<Supplier>>(`/api/v1/suppliers/${id}`, payload, token)
    return res.data
  },

  softDelete: async (id: number, token?: string): Promise<{ id: number }> => {
    const res = await api.delete<ApiEnvelope<{ id: number }>>(`/api/v1/suppliers/${id}`, token)
    return res.data
  },
}
