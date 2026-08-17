import type { ApiEnvelope } from '../types/auth'
import type { Customer, CustomerFilter } from '../types/customer'
import { api } from '../lib/api'

export const customersApi = {
  list: async (filter?: CustomerFilter, token?: string): Promise<Customer[]> => {
    const q = [] as string[]
    if (filter?.search) q.push(`search=${encodeURIComponent(filter.search)}`)
    if (filter?.active !== undefined) q.push(`active=${filter.active}`)
    const path = `/api/v1/customers${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Customer[]>>(path, token)
    return res.data
  },

  getById: async (id: number, token?: string): Promise<Customer> => {
    const res = await api.get<ApiEnvelope<Customer>>(`/api/v1/customers/${id}`, token)
    return res.data
  },

  create: async (payload: Partial<Customer>, token?: string): Promise<Customer> => {
    const res = await api.post<ApiEnvelope<Customer>>(`/api/v1/customers`, payload, token)
    return res.data
  },

  update: async (id: number, payload: Partial<Customer>, token?: string): Promise<Customer> => {
    const res = await api.put<ApiEnvelope<Customer>>(`/api/v1/customers/${id}`, payload, token)
    return res.data
  },

  softDelete: async (id: number, token?: string): Promise<{ id: number }> => {
    const res = await api.delete<ApiEnvelope<{ id: number }>>(`/api/v1/customers/${id}`, token)
    return res.data
  },

  remove: async (id: number, token?: string): Promise<{ id: number }> => {
    return customersApi.softDelete(id, token)
  },
}
