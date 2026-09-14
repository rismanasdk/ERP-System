import type { ApiEnvelope } from '../types/auth'
import type { Sale, SaleFilter, CreateSaleInput, SaleFulfillment, FulfillSaleInput, SalesPayment, PaymentSummary } from '../types/sale'
import { api } from '../lib/api'

export const salesApi = {
  list: async (filter?: SaleFilter, token?: string): Promise<Sale[]> => {
    const q: string[] = []
    if (filter?.branch_id !== undefined) q.push(`branch_id=${filter.branch_id}`)
    const path = `/api/v1/sales${q.length ? `?${q.join('&')}` : ''}`
    const res = await api.get<ApiEnvelope<Sale[]>>(path, token)
    return res.data ?? []
  },

  getById: async (id: number, token?: string): Promise<Sale> => {
    const res = await api.get<ApiEnvelope<Sale>>(`/api/v1/sales/${id}`, token)
    return res.data
  },

  create: async (payload: CreateSaleInput, token?: string): Promise<{ id: number; sale_number: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; sale_number: string }>>(`/api/v1/sales`, payload, token)
    return res.data
  },

  complete: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/sales/${id}/complete`, undefined, token)
    return res.data
  },

  cancel: async (id: number, token?: string): Promise<{ id: number; status: string }> => {
    const res = await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/sales/${id}/cancel`, undefined, token)
    return res.data
  },

  confirm: async (id: number, token?: string) => (await api.post<ApiEnvelope<{ id: number; status: string }>>(`/api/v1/sales/${id}/confirm`, undefined, token)).data,
  fulfill: async (id: number, payload: FulfillSaleInput, token?: string) => (await api.post<ApiEnvelope<{ id: number }>>(`/api/v1/sales/${id}/fulfill`, payload, token)).data,
  listFulfillments: async (id: number, token?: string): Promise<SaleFulfillment[]> => (await api.get<ApiEnvelope<SaleFulfillment[]>>(`/api/v1/sales/${id}/fulfillments`, token)).data ?? [],
  listPayments: async (id: number, token?: string): Promise<SalesPayment[]> => (await api.get<ApiEnvelope<SalesPayment[]>>(`/api/v1/sales/${id}/payments`, token)).data ?? [],
  paymentSummary: async (id: number, token?: string): Promise<PaymentSummary> => (await api.get<ApiEnvelope<PaymentSummary>>(`/api/v1/sales/${id}/payment-summary`, token)).data,
  createPayment: async (id: number, payload: { amount: number; payment_method: string; reference_number?: string; notes?: string }, token?: string): Promise<{ id: number }> => (await api.post<ApiEnvelope<{ id: number }>>(`/api/v1/sales/${id}/payments`, payload, token)).data,
}
