import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { ConfirmDialogProvider } from '../utils/confirmUtils'
import { SalesPage } from '../pages/SalesPage'
import { ApiError } from '../lib/api'

vi.mock('../services/sales', () => ({
  salesApi: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    complete: vi.fn(),
    cancel: vi.fn(),
    confirm: vi.fn(),
    fulfill: vi.fn(),
    listFulfillments: vi.fn(),
    listPayments: vi.fn(),
    paymentSummary: vi.fn(),
    createPayment: vi.fn(),
  },
}))

vi.mock('../services/branches', () => ({
  branchesApi: {
    list: vi.fn(),
    getById: vi.fn(),
  },
}))

vi.mock('../services/products', () => ({
  productsApi: {
    list: vi.fn(),
  },
}))

vi.mock('../services/customers', () => ({
  customersApi: { getById: vi.fn(), list: vi.fn() },
}))

import { salesApi } from '../services/sales'
import { branchesApi } from '../services/branches'
import { productsApi } from '../services/products'
import { customersApi } from '../services/customers'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('SalesPage', () => {
  it('renders sale data with branch names', async () => {
    ;(salesApi.getById as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({})
    ;(salesApi.listFulfillments as unknown as ReturnType<typeof vi.fn>).mockResolvedValue([])
    ;(salesApi.paymentSummary as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({ order_total: 0, paid_amount: 0, remaining_amount: 0, payment_status: 'UNPAID' })
    ;(salesApi.listPayments as unknown as ReturnType<typeof vi.fn>).mockResolvedValue([])
    const listMock = salesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([
      {
        id: 1,
        branch_id: 2,
        sale_number: 'SALE-001',
        status: 'DRAFT',
        total_amount: 150000,
        created_by: 3,
        created_at: '2024-01-01T00:00:00Z',
      },
    ])

    const branchMock = branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'Main Branch', code: 'MBR' })
    const customerMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    customerMock.mockResolvedValue([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <SalesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('SALE-001')).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('Main Branch')).toBeInTheDocument())
  })

  it('allows creating a sale', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['sales.create', 'sales.read'] }))
    ;(salesApi.getById as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({})
    ;(salesApi.listFulfillments as unknown as ReturnType<typeof vi.fn>).mockResolvedValue([])
    ;(salesApi.paymentSummary as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({ order_total: 0, paid_amount: 0, remaining_amount: 0, payment_status: 'UNPAID' })
    ;(salesApi.listPayments as unknown as ReturnType<typeof vi.fn>).mockResolvedValue([])

    let salesRows: Array<{
      id: number
      branch_id: number
      sale_number: string
      status: string
      total_amount: number
      created_by: number
      created_at: string
    }> = []

    const listMock = salesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockImplementation(async () => salesRows)

    const createMock = salesApi.create as unknown as ReturnType<typeof vi.fn>
    createMock.mockImplementation(async () => {
      salesRows = [{
        id: 10,
        branch_id: 2,
        sale_number: 'SALE-010',
        status: 'DRAFT',
        total_amount: 25000,
        created_by: 1,
        created_at: '2024-01-02T00:00:00Z',
      }]
      return { id: 10, sale_number: 'SALE-010' }
    })

    const productListMock = productsApi.list as unknown as ReturnType<typeof vi.fn>
    productListMock.mockResolvedValueOnce([
      { id: 5, sku: 'P-005', name: 'Widget', purchase_price: 1000, selling_price: 2000, is_active: true },
    ])
    const customerListMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    customerListMock.mockResolvedValueOnce([{ id: 3, code: 'C-003', name: 'Customer One', is_active: true }])

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValueOnce([
      { id: 2, code: 'MBR', name: 'Main Branch', is_active: true },
    ])

    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <SalesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no sales found/i)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /create/i }))

    await waitFor(() => expect(screen.getByLabelText(/branch/i)).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText(/branch/i), '2')
    await user.selectOptions(screen.getByLabelText(/customer/i), '3')
    await user.click(screen.getByRole('button', { name: /add item/i }))
    await user.selectOptions(screen.getByLabelText(/product/i), '5')
    await user.clear(screen.getByLabelText(/quantity/i))
    await user.type(screen.getByLabelText(/quantity/i), '2')
    await user.clear(screen.getByLabelText(/unit price/i))
    await user.type(screen.getByLabelText(/unit price/i), '12500')
    await user.click(screen.getByRole('button', { name: /create sale/i }))

    await waitFor(() => expect(screen.getByText('SALE-010')).toBeInTheDocument())
  })

  it('asks for confirmation before completing a sale', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['sales.complete', 'sales.read'] }))

    let salesRows: Array<{
      id: number
      branch_id: number
      sale_number: string
      status: string
      total_amount: number
      created_by: number
      created_at: string
    }> = [{
      id: 9,
      branch_id: 2,
      sale_number: 'SALE-009',
      status: 'DRAFT',
      total_amount: 150000,
      created_by: 1,
      created_at: '2024-01-01T00:00:00Z',
    }]

    const listMock = salesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockImplementation(async () => salesRows)

    const completeMock = salesApi.complete as unknown as ReturnType<typeof vi.fn>
    completeMock.mockImplementation(async () => {
      salesRows = [{
        id: 9,
        branch_id: 2,
        sale_number: 'SALE-009',
        status: 'COMPLETED',
        total_amount: 150000,
        created_by: 1,
        created_at: '2024-01-01T00:00:00Z',
      }]
      return { id: 9, status: 'COMPLETED' }
    })

    const branchMock = branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'Main Branch', code: 'MBR' })

    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <SalesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('SALE-009')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /complete/i }))
    await user.click(screen.getByRole('button', { name: /confirm/i }))

    await waitFor(() => expect(screen.getByText('COMPLETED')).toBeInTheDocument())
  })

  it('shows API error when the load fails with 403', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = salesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <SalesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have access to sales/i)).toBeInTheDocument())
  })

  it('shows payment summary, history, and adds a payment', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['sales.read', 'sales.payment.read', 'sales.payment.create'] }))
    const sale = { id: 15, branch_id: 2, customer_id: 3, sale_number: 'SALE-015', status: 'CONFIRMED', total_amount: 1000000, created_by: 1, created_at: '2024-01-01T00:00:00Z', items: [] }
    const listMock = salesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([sale])
    const detailMock = salesApi.getById as unknown as ReturnType<typeof vi.fn>
    detailMock.mockResolvedValue(sale)
    const summaryMock = salesApi.paymentSummary as unknown as ReturnType<typeof vi.fn>
    summaryMock.mockResolvedValue({ order_total: 1000000, paid_amount: 400000, remaining_amount: 600000, payment_status: 'PARTIALLY_PAID' })
    const historyMock = salesApi.listPayments as unknown as ReturnType<typeof vi.fn>
    historyMock.mockResolvedValue([{ id: 1, sales_order_id: 15, amount: 400000, payment_method: 'BANK_TRANSFER', reference_number: 'TRX-001', created_by: 1, created_by_name: 'Risman' }])
    const createMock = salesApi.createPayment as unknown as ReturnType<typeof vi.fn>
    createMock.mockResolvedValue({ id: 2 })
    const branchMock = branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValue({ id: 2, name: 'Main Branch', code: 'MBR' })
    const customerMock = customersApi.getById as unknown as ReturnType<typeof vi.fn>
    customerMock.mockResolvedValue({ id: 3, code: 'C-003', name: 'Customer One', is_active: true })
    const user = userEvent.setup()
    render(<MemoryRouter><AuthProvider><ConfirmDialogProvider><SalesPage /></ConfirmDialogProvider></AuthProvider></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('SALE-015')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /view/i }))
    await waitFor(() => expect(screen.getByText('PARTIALLY_PAID')).toBeInTheDocument())
    expect(screen.getByText(/payment summary/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText(/TRX-001/)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /add payment/i }))
    await user.type(screen.getByLabelText(/amount/i), '600000')
    await user.click(screen.getByRole('button', { name: /save payment/i }))
    await waitFor(() => expect(createMock).toHaveBeenCalledWith(15, { amount: 600000, payment_method: 'CASH', reference_number: undefined, notes: undefined }, undefined))
  })
})
