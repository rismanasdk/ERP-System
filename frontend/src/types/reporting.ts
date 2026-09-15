export type DailySalesSummary = {
  date: string
  total_sales: number
  total_transactions: number
  total_items_sold: number
  total_revenue: number
}

export type SalesReport = {
  total_sales: number
  total_transactions: number
  total_items_sold: number
  total_revenue: number
  total_cancelled_sales?: number
  cancelled_sales_value?: number
  daily_summary?: DailySalesSummary[]
}

export type PurchaseReport = {
  total_purchases?: number
  total_purchase_transactions?: number
  total_purchased_items?: number
  total_purchase_amount?: number
  completed_purchases?: number
  cancelled_purchases?: number
  daily_summary?: Array<{
    date: string
    total_purchases: number
    total_transactions: number
    total_purchased_items: number
    total_purchase_amount: number
    completed_purchases: number
    cancelled_purchases: number
  }>
}

export type InventoryReport = {
  total_inventory_records?: number
  total_quantity?: number
  stock_movement_summary?: {
    in_quantity?: number
    out_quantity?: number
    adjustment_quantity?: number
  }
  low_stock_products?: Array<{
    product_id: number
    branch_id: number
    quantity: number
    name?: string
  }>
  rows?: Array<{
    branch_id: number
    branch_name?: string
    product_id: number
    product_name?: string
    sku?: string
    category_name?: string
    current_stock: number
    minimum_stock: number
    stock_status?: string
  }>
}

export type PaymentReport = {
  total_payments?: number
  total_payment_amount?: number
  method_breakdown?: Record<string, number>
  method_total_breakdown?: Record<string, number>
  rows?: Array<{
    id: number
    sale_number?: string
    customer_name?: string
    branch_name?: string
    amount: number
    payment_method: string
    reference_number?: string
    paid_at: string
  }>
}

export type SalesReportFilter = {
  start_date?: string
  end_date?: string
  branch_id?: number
}

export type ReportsTab = 'sales' | 'purchases' | 'inventory' | 'payments'
