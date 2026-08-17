import type { ApiEnvelope } from '../types/auth'
import type { Branch } from '../types/auth'
import { api } from '../lib/api'

export const branchesApi = {
  list: async (active?: boolean, token?: string): Promise<Branch[]> => {
    const q: string[] = []
    if (active !== undefined) q.push(`active=${active}`)
    const path = `/api/v1/branches${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Branch[]>>(path, token)
    return res.data ?? []
  },

  getById: async (id: number, token?: string): Promise<Branch> => {
    const res = await api.get<ApiEnvelope<Branch>>(`/api/v1/branches/${id}`, token)
    return res.data
  },

  create: async (payload: Partial<Branch>, token?: string): Promise<Branch> => {
    const res = await api.post<ApiEnvelope<Branch>>('/api/v1/branches', payload, token)
    return res.data
  },

  update: async (id: number, payload: Partial<Branch>, token?: string): Promise<Branch> => {
    const res = await api.put<ApiEnvelope<Branch>>(`/api/v1/branches/${id}`, payload, token)
    return res.data
  },
}
