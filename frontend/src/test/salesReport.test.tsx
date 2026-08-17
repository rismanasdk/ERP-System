import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { SalesReportPage } from '../pages/SalesReportPage'
import { ApiError } from '../lib/api'

vi.mock('../services/reports', () => ({
  salesReportsApi: {
    get: vi.fn(),
  },
}))

vi.mock('../services/branches', () => ({
  branchesApi: {
    list: vi.fn(),
  },
}))

import { salesReportsApi } from '../services/reports'
import { branchesApi } from '../services/branches'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('SalesReportPage', () => {
  it('shows initial loading state', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockReturnValue(new Promise(() => {}))

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByText(/sales report/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText(/apply/i)).toBeInTheDocument())
  })

  it('renders successful sales report summary and daily table', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockResolvedValue({
      total_sales: 1500000,
      total_transactions: 12,
      total_items_sold: 40,
      total_revenue: 1500000,
      total_cancelled_sales: 0,
      cancelled_sales_value: 0,
      daily_summary: [
        {
          date: '2026-08-01',
          total_sales: 500000,
          total_transactions: 4,
          total_items_sold: 15,
          total_revenue: 500000,
        },
      ],
    })

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getAllByText('Total Sales').length).toBeGreaterThan(0))
    await waitFor(() => expect(screen.getByText(/Rp\s*1\.500\.000/i)).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('Daily Sales Summary')).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('01/08/2026')).toBeInTheDocument())
  })

  it('sends start_date, end_date, and branch_id when filtering', async () => {
    localStorage.setItem('erp_access_token', 'test-token')

    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockResolvedValue({
      total_sales: 0,
      total_transactions: 0,
      total_items_sold: 0,
      total_revenue: 0,
      total_cancelled_sales: 0,
      cancelled_sales_value: 0,
      daily_summary: [],
    })

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    const startDateInput = await screen.findByLabelText(/start date/i)
    const endDateInput = await screen.findByLabelText(/end date/i)
    const branchSelect = await screen.findByLabelText(/branch/i)

    await user.clear(startDateInput)
    await user.type(startDateInput, '2026-01-01')
    await user.clear(endDateInput)
    await user.type(endDateInput, '2026-01-31')
    await user.selectOptions(branchSelect, '2')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(getMock).toHaveBeenLastCalledWith(
        {
          start_date: '2026-01-01',
          end_date: '2026-01-31',
          branch_id: 2,
        },
        'test-token',
      )
    })
  })

  it('shows empty state when no report data exists', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockResolvedValue({
      total_sales: 0,
      total_transactions: 0,
      total_items_sold: 0,
      total_revenue: 0,
      total_cancelled_sales: 0,
      cancelled_sales_value: 0,
      daily_summary: [],
    })

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no sales data found for this period/i)).toBeInTheDocument())
  })

  it('shows 401 session expired state', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockRejectedValue(new ApiError(401, 'Session expired', 'UNAUTHORIZED'))

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/session expired/i)).toBeInTheDocument())
  })

  it('shows 403 permission denied state', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockRejectedValue(new ApiError(403, 'Forbidden', 'FORBIDDEN'))

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/do not have access to sales reports/i)).toBeInTheDocument())
  })

  it('shows generic API error state', async () => {
    const getMock = salesReportsApi.get as unknown as ReturnType<typeof vi.fn>
    getMock.mockRejectedValue(new ApiError(500, 'Server error', 'INTERNAL_SERVER_ERROR'))

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValue([{ id: 2, name: 'Main Branch', code: 'MBR', is_active: true }])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SalesReportPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/server error/i)).toBeInTheDocument())
  })
})
