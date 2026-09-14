export type Sale = {
  id: number
  branch_id: number
  customer_id: number
  sale_number: string
  status: 'DRAFT' | 'CONFIRMED' | 'PARTIALLY_FULFILLED' | 'FULFILLED' | 'COMPLETED' | 'CANCELLED' | string
  total_amount: number
  notes?: string | null
  created_by: number
  created_at?: string
  updated_at?: string
  items?: SaleItem[]
}

export type SaleFulfillment = { id: number; sales_order_id: number; branch_id: number; fulfilled_by: number; fulfilled_by_name?: string; fulfilled_at?: string; notes?: string | null; items: { id: number; sales_order_item_id: number; product_id: number; quantity_fulfilled: number }[] }

export type SaleFilter = {
  branch_id?: number
}

export type SaleItem = {
  id: number
  sale_id: number
  product_id: number
  quantity: number
  unit_price: number
  subtotal: number
  fulfilled_quantity: number
  created_at?: string
  updated_at?: string
}

export type CreateSaleItemInput = {
  product_id: number
  quantity: number
  unit_price: number
}

export type CreateSaleInput = {
  customer_id: number
  branch_id: number
  notes?: string | null
  items: CreateSaleItemInput[]
}

export type FulfillSaleInput = { notes?: string; items: { sales_order_item_id: number; quantity_fulfilled: number }[] }
export type SalesPayment = { id: number; sales_order_id: number; amount: number; payment_method: 'CASH' | 'BANK_TRANSFER' | 'OTHER' | 'QRIS' | string; reference_number?: string | null; paid_at?: string; notes?: string | null; created_by: number; created_by_name?: string; created_at?: string }
export type PaymentSummary = { order_total: number; paid_amount: number; remaining_amount: number; payment_status: 'UNPAID' | 'PARTIALLY_PAID' | 'PAID' | string }

export type SaleWithItems = Sale & {
  items?: SaleItem[]
}
