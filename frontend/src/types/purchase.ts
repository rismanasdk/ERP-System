export type Purchase = {
  id: number
  branch_id: number
  supplier_id: number
  purchase_number: string
  status: 'DRAFT' | 'COMPLETED' | 'CANCELLED' | string
  total_amount: number
  notes?: string | null
  created_by: number
  created_at?: string
  updated_at?: string
}

export type PurchaseFilter = {
  branch_id?: number
  supplier_id?: number
  status?: string
}

export type PurchaseItem = {
  id: number
  purchase_id: number
  product_id: number
  quantity: number
  unit_cost: number
  subtotal: number
  created_at?: string
  updated_at?: string
}

export type CreatePurchaseItemInput = {
  product_id: number
  quantity: number
  unit_cost: number
}

export type CreatePurchaseInput = {
  branch_id: number
  supplier_id: number
  notes?: string | null
  items: CreatePurchaseItemInput[]
}
