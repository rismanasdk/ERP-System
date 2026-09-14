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
  items?: PurchaseItem[]
}

export type PurchaseReceipt = {
  id: number
  purchase_order_id: number
  branch_id: number
  received_by: number
  received_by_name?: string
  received_at?: string
  notes?: string | null
  items: PurchaseReceiptItem[]
}

export type PurchaseReceiptItem = {
  id: number
  purchase_order_item_id: number
  product_id: number
  quantity_received: number
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
  received_quantity: number
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
