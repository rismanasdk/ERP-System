import type { ApiEnvelope } from '../types/auth'
import type { User } from '../types/auth'
import { api } from '../lib/api'
import { readStoredAccessToken } from './authSession'

export const usersApi = {
  list: async (token?: string): Promise<User[]> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<User[]>>('/api/v1/users', t)
    return res.data ?? []
  },

  getById: async (id: number, token?: string): Promise<User> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<User>>(`/api/v1/users/${id}`, t)
    return res.data
  },

  create: async (payload: Partial<User> & { password?: string; branch_ids?: number[] }, token?: string): Promise<User> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.post<ApiEnvelope<User>>('/api/v1/users', payload, t)
    return res.data
  },

  update: async (id: number, payload: Partial<User> & { password?: string; branch_ids?: number[] }, token?: string): Promise<User> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.put<ApiEnvelope<User>>(`/api/v1/users/${id}`, payload, t)
    return res.data
  },

  remove: async (id: number, token?: string): Promise<void> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    await api.delete<ApiEnvelope<null>>(`/api/v1/users/${id}`, t)
  },
}

export type { User }
