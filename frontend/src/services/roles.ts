import type { ApiEnvelope } from '../types/auth'
import { api } from '../lib/api'
import { readStoredAccessToken } from './authSession'

export const rolesApi = {
  list: async (token?: string): Promise<string[]> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<unknown>>('/api/v1/roles', t)
    const data = res?.data as unknown
    if (!data) return []
    if (Array.isArray(data)) {
      return data.map((r) => {
        if (typeof r === 'string') return r
        if (r && typeof r === 'object') {
          const obj = r as Record<string, unknown>
          const name = obj['name']
          return typeof name === 'string' ? name : ''
        }
        return ''
      }).filter(Boolean)
    }
    return []
  },

  // listAll returns full role objects as exposed by backend, including permissions when available
  all: async (token?: string): Promise<Array<{ id: number; name: string; description?: string; permissions?: string[] }>> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<unknown>>('/api/v1/roles', t)
    const data = res?.data as unknown
    if (!data) return []
    if (Array.isArray(data)) {
      return data.map((item) => {
        if (typeof item === 'string') {
          return { id: 0, name: item }
        }
        if (item && typeof item === 'object') {
          const obj = item as Record<string, unknown>
          return {
            id: typeof obj['id'] === 'number' ? obj['id'] as number : 0,
            name: typeof obj['name'] === 'string' ? obj['name'] as string : '',
            description: typeof obj['description'] === 'string' ? obj['description'] as string : undefined,
            permissions: Array.isArray(obj['permissions']) ? (obj['permissions'] as unknown[]).map((p) => String(p)) : undefined,
          }
        }
        return { id: 0, name: '' }
      }).filter((r) => r.name)
    }
    return []
  },

  get: async (id: number, token?: string): Promise<{ id: number; name: string; description?: string; permissions?: string[] } | null> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<unknown>>(`/api/v1/roles/${id}`, t)
    const data = res?.data as unknown
    if (!data || typeof data !== 'object') return null
    const obj = data as Record<string, unknown>
    return {
      id: typeof obj['id'] === 'number' ? obj['id'] as number : 0,
      name: typeof obj['name'] === 'string' ? obj['name'] as string : '',
      description: typeof obj['description'] === 'string' ? obj['description'] as string : undefined,
      permissions: Array.isArray(obj['permissions']) ? (obj['permissions'] as unknown[]).map((p) => String(p)) : undefined,
    }
  },

  permissions: async (token?: string): Promise<string[]> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<unknown>>('/api/v1/permissions', t)
    const data = res?.data as unknown
    if (!data) return []
    if (Array.isArray(data)) {
      return data.map((p) => {
        if (typeof p === 'string') return p
        if (p && typeof p === 'object') {
          const obj = p as Record<string, unknown>
          const name = obj['name']
          return typeof name === 'string' ? name : ''
        }
        return ''
      }).filter(Boolean)
    }
    return []
  },
}

export type { }
