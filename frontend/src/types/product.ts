export type Product = {
  id: number
  sku: string
  barcode?: string | null
  name: string
  description?: string | null
  category?: string | null
  category_id?: number | null
  unit?: string | null
  purchase_price: number
  selling_price: number
  minimum_stock: number
  version: number
  is_active: boolean
  created_at?: string
  updated_at?: string
  deleted_at?: string | null
}

export type ProductFilter = {
  search?: string
  active?: boolean
  category_id?: number
}
