export type Customer = {
  id: number
  code: string
  name: string
  phone?: string | null
  email?: string | null
  address?: string | null
  tax_id?: string | null
  is_active: boolean
  created_at?: string
  updated_at?: string
  deleted_at?: string | null
}

export type CustomerFilter = {
  search?: string
  active?: boolean
}
