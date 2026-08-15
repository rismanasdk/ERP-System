export type DashboardMetricBlock = {
  today_amount: number
  today_transactions: number
  month_amount: number
  month_transactions: number
}

export type DashboardMasterData = {
  products: number
  customers: number
  suppliers: number
}

export type DashboardInventory = {
  total_items: number
}

export type DashboardSummary = {
  sales: DashboardMetricBlock
  purchases: DashboardMetricBlock
  master_data: DashboardMasterData
  inventory: DashboardInventory
}
