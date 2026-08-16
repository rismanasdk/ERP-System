export type Sale = {
  id: number
  branch_id: number
  sale_number: string
  status: 'DRAFT' | 'COMPLETED' | 'CANCELLED' | string
  total_amount: number
  notes?: string | null
  created_by: number
  created_at?: string
  updated_at?: string
}

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
  created_at?: string
  updated_at?: string
}

export type CreateSaleItemInput = {
  product_id: number
  quantity: number
  unit_price: number
}

export type CreateSaleInput = {
  branch_id: number
  notes?: string | null
  items: CreateSaleItemInput[]
}
