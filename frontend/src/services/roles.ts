import type { ApiEnvelope, Permission, Role } from '../types/auth'
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

  all: async (token?: string): Promise<Role[]> => {
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
            user_count: typeof obj['user_count'] === 'number' ? obj['user_count'] as number : 0,
          } satisfies Role
        }
        return { id: 0, name: '' } satisfies Role
      }).filter((r) => r.name)
    }
    return []
  },

  get: async (id: number, token?: string): Promise<Role | null> => {
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
      user_count: typeof obj['user_count'] === 'number' ? obj['user_count'] as number : 0,
    } satisfies Role
  },

  permissions: async (token?: string): Promise<Permission[]> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.get<ApiEnvelope<unknown>>('/api/v1/permissions', t)
    const data = res?.data as unknown
    if (!data) return []
    if (Array.isArray(data)) {
      const normalized: Array<Permission | null> = data.map((p, index): Permission | null => {
        if (typeof p === 'string') return { id: index, name: p }
        if (p && typeof p === 'object') {
          const obj = p as Record<string, unknown>
          const name = obj['name']
          return typeof name === 'string' ? { id: typeof obj['id'] === 'number' ? obj['id'] as number : index, name, description: typeof obj['description'] === 'string' ? obj['description'] as string : undefined } : null
        }
        return null
      })
      return normalized.filter((permission): permission is Permission => permission !== null)
    }
    return []
  },

  create: async (payload: { name: string; description: string; permissions: string[] }, token?: string): Promise<Role> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.post<ApiEnvelope<Role>>('/api/v1/roles', payload, t)
    return res.data
  },

  update: async (id: number, payload: { name: string; description: string; permissions: string[] }, token?: string): Promise<Role> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    const res = await api.put<ApiEnvelope<Role>>(`/api/v1/roles/${id}`, payload, t)
    return res.data
  },

  remove: async (id: number, token?: string): Promise<void> => {
    const t = token ?? readStoredAccessToken() ?? undefined
    await api.delete<ApiEnvelope<{ id: number }>>(`/api/v1/roles/${id}`, t)
  },
}
