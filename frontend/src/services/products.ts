import type { ApiEnvelope } from '../types/auth' // pastikan diimport
import type { Product, ProductFilter } from '../types/product'
import { api } from '../lib/api'

export type ProductUpdatePayload = Partial<Product> & { expected_version: number }

export const productsApi = {
  list: async (filter?: ProductFilter, token?: string): Promise<Product[]> => {
    const q = [] as string[]
    if (filter?.search) q.push(`search=${encodeURIComponent(filter.search)}`)
    if (filter?.active !== undefined) q.push(`active=${filter.active}`)
    if (filter?.category_id !== undefined) q.push(`category_id=${filter.category_id}`)
    const path = `/api/v1/products${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Product[]>>(path, token)
    return res.data
  },

  getById: async (id: number, token?: string): Promise<Product> => {
    const res = await api.get<ApiEnvelope<Product>>(`/api/v1/products/${id}`, token)
    return res.data
  },

  create: async (payload: Partial<Product>, token?: string): Promise<Product> => {
    const res = await api.post<ApiEnvelope<Product>>(`/api/v1/products`, payload, token)
    return res.data
  },

  update: async (id: number, payload: ProductUpdatePayload, token?: string): Promise<Product> => {
    const res = await api.put<ApiEnvelope<Product>>(`/api/v1/products/${id}`, payload, token)
    return res.data
  },

  softDelete: async (id: number, token?: string): Promise<{ id: number }> => {
    const res = await api.delete<ApiEnvelope<{ id: number }>>(`/api/v1/products/${id}`, token)
    return res.data
  },
}