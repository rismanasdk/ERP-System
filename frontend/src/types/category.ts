export type ProductCategory = {
  id: number
  name: string
  description?: string | null
  is_active: boolean
  product_count: number
  created_at?: string
  updated_at?: string
}

export type CategoryFilter = { search?: string; active?: boolean }