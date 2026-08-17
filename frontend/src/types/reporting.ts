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
  total_cancelled_sales: number
  cancelled_sales_value: number
  daily_summary?: DailySalesSummary[]
}

export type SalesReportFilter = {
  start_date?: string
  end_date?: string
  branch_id?: number
}
