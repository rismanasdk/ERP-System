import type { ApiEnvelope } from '../types/auth'
import type { CategoryFilter, ProductCategory } from '../types/category'
import { api } from '../lib/api'

export type CategoryUpdatePayload = Partial<ProductCategory> & { expected_version: number }

export const categoriesApi = {
  list: async (filter?: CategoryFilter, token?: string): Promise<ProductCategory[]> => {
    const query: string[] = []
    if (filter?.search) query.push(`search=${encodeURIComponent(filter.search)}`)
    if (filter?.active !== undefined) query.push(`active=${filter.active}`)
    const suffix = query.length ? `?${query.join('&')}` : ''
    const response = await api.get<ApiEnvelope<ProductCategory[]>>(`/api/v1/product-categories${suffix}`, token)
    return response.data ?? []
  },
  get: async (id: number, token?: string): Promise<ProductCategory> => (await api.get<ApiEnvelope<ProductCategory>>(`/api/v1/product-categories/${id}`, token)).data,
  create: async (payload: Partial<ProductCategory>, token?: string): Promise<ProductCategory> => (await api.post<ApiEnvelope<ProductCategory>>('/api/v1/product-categories', payload, token)).data,
  update: async (id: number, payload: CategoryUpdatePayload, token?: string): Promise<ProductCategory> => (await api.put<ApiEnvelope<ProductCategory>>(`/api/v1/product-categories/${id}`, payload, token)).data,
  remove: async (id: number, token?: string): Promise<void> => { await api.delete<ApiEnvelope<{ id: number }>>(`/api/v1/product-categories/${id}`, token) },
}