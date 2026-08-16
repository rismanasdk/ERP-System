import type { ApiEnvelope } from '../types/auth'
import type { Inventory, InventoryFilter } from '../types/inventory'
import { api } from '../lib/api'

export const inventoryApi = {
    list: async (filter?: InventoryFilter, token?: string): Promise<Inventory[]> => {
      const q: string[] = []
      if (filter?.branch_id !== undefined) q.push(`branch_id=${filter.branch_id}`)
      if (filter?.product_id !== undefined) q.push(`product_id=${filter.product_id}`)
      const path = `/api/v1/inventory${q.length ? `?${q.join('&')}` : ''}`
      const res = await api.get<ApiEnvelope<Inventory[]>>(path, token)
      return res.data ?? []
    },

  getById: async (id: number, token?: string): Promise<Inventory> => {
    const res = await api.get<ApiEnvelope<Inventory>>(`/api/v1/inventory/${id}`, token)
    return res.data ?? {}
  },

  create: async (payload: { product_id: number; branch_id: number; quantity: number }, token?: string): Promise<{ id: number }> => {
    const res = await api.post<ApiEnvelope<{ id: number }>>(`/api/v1/inventory`, payload, token)
    return res.data
  },

  adjust: async (
    inventoryId: number,
    payload: { movement_type: string; quantity_delta: number; reference_type?: string; reference_id?: number },
    token?: string,
  ): Promise<{ movement_id: number }> => {
    const res = await api.post<ApiEnvelope<{ movement_id: number }>>(`/api/v1/inventory/${inventoryId}/adjust`, payload, token)
    return res.data
  },
}
