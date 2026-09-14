export type Inventory = {
  id: number
  product_id: number
  branch_id: number
  quantity: number
  created_at?: string
  updated_at?: string
}

export type StockMovement = {
  id: number
  product_id: number
  branch_id: number
  movement_type: string
  quantity_delta: number
  reference_type?: string | null
  reference_id?: number | null
  actor_user_id?: number | null
  actor_user_name?: string | null
  metadata?: Record<string, unknown> | null
  created_at?: string
}

export type InventoryFilter = {
  branch_id?: number
  product_id?: number
}

export type StockTransfer = {
  id: number
  source_branch_id: number
  source_branch_name?: string
  destination_branch_id: number
  destination_branch_name?: string
  product_id: number
  quantity: number
  status: string
  notes?: string | null
  created_by: number
  created_by_name?: string
  created_at?: string
  completed_at?: string
}
