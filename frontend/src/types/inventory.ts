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
  created_at?: string
}

export type InventoryFilter = {
  branch_id?: number
  product_id?: number
}
