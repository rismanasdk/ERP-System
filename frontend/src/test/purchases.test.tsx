import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { ConfirmDialogProvider } from '../utils/confirmUtils'
import { PurchasesPage } from '../pages/PurchasesPage'
import { ApiError } from '../lib/api'

vi.mock('../services/purchases', () => ({
  purchasesApi: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    complete: vi.fn(),
    cancel: vi.fn(),
  },
}))

vi.mock('../services/branches', () => ({
  branchesApi: {
    list: vi.fn(),
    getById: vi.fn(),
  },
}))

vi.mock('../services/suppliers', () => ({
  suppliersApi: {
    list: vi.fn(),
    getById: vi.fn(),
  },
}))

vi.mock('../services/products', () => ({
  productsApi: {
    list: vi.fn(),
  },
}))

import { purchasesApi } from '../services/purchases'
import { branchesApi } from '../services/branches'
import { suppliersApi } from '../services/suppliers'
import { productsApi } from '../services/products'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('PurchasesPage', () => {
  it('renders purchase data with supplier and branch names', async () => {
    const listMock = purchasesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([
      {
        id: 1,
        branch_id: 2,
        supplier_id: 7,
        purchase_number: 'PO-001',
        status: 'DRAFT',
        total_amount: 150000,
        created_at: '2024-01-01T00:00:00Z',
      },
    ])

    const branchMock = branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'Main Branch', code: 'MBR' })

    const supplierMock = suppliersApi.getById as unknown as ReturnType<typeof vi.fn>
    supplierMock.mockResolvedValueOnce({ id: 7, code: 'SUP', name: 'Supply Co', is_active: true })

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <PurchasesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('PO-001')).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('Main Branch')).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('Supply Co')).toBeInTheDocument())
  })

  it('allows creating a purchase', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['purchases.create'] }))

    const listMock = purchasesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])

    const createMock = purchasesApi.create as unknown as ReturnType<typeof vi.fn>
    createMock.mockResolvedValueOnce({ id: 10, purchase_number: 'PO-010' })

    const supplierListMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    supplierListMock.mockResolvedValueOnce([
      { id: 7, code: 'SUP', name: 'Supply Co', is_active: true },
    ])

    const branchListMock = branchesApi.list as unknown as ReturnType<typeof vi.fn>
    branchListMock.mockResolvedValueOnce([
      { id: 2, code: 'MBR', name: 'Main Branch', is_active: true },
    ])

    const productListMock = productsApi.list as unknown as ReturnType<typeof vi.fn>
    productListMock.mockResolvedValueOnce([
      { id: 5, sku: 'P-005', name: 'Widget', purchase_price: 1000, selling_price: 2000, is_active: true },
    ])

    listMock.mockResolvedValueOnce([
      {
        id: 10,
        branch_id: 2,
        supplier_id: 7,
        purchase_number: 'PO-010',
        status: 'DRAFT',
        total_amount: 25000,
        created_at: '2024-01-02T00:00:00Z',
      },
    ])

    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <PurchasesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no purchases found/i)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /create/i }))

    await waitFor(() => expect(screen.getByLabelText(/branch/i)).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText(/branch/i), '2')
    await user.selectOptions(screen.getByLabelText(/supplier/i), '7')
    await user.click(screen.getByRole('button', { name: /add item/i }))
    await waitFor(() => expect(screen.getByDisplayValue('1')).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText(/product/i), '5')
    await user.clear(screen.getByLabelText(/quantity/i))
    await user.type(screen.getByLabelText(/quantity/i), '2')
    await user.clear(screen.getByLabelText(/unit cost/i))
    await user.type(screen.getByLabelText(/unit cost/i), '12500')
    await user.click(screen.getByRole('button', { name: /create purchase/i }))

    await waitFor(() => expect(screen.getByText('PO-010')).toBeInTheDocument())
  })

  it('asks for confirmation before completing a purchase', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['purchases.complete'] }))

    const listMock = purchasesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([
      {
        id: 9,
        branch_id: 2,
        supplier_id: 7,
        purchase_number: 'PO-009',
        status: 'DRAFT',
        total_amount: 150000,
        created_at: '2024-01-01T00:00:00Z',
      },
    ])

    const completeMock = purchasesApi.complete as unknown as ReturnType<typeof vi.fn>
    completeMock.mockResolvedValueOnce({ id: 9, status: 'COMPLETED' })

    listMock.mockResolvedValueOnce([
      {
        id: 9,
        branch_id: 2,
        supplier_id: 7,
        purchase_number: 'PO-009',
        status: 'COMPLETED',
        total_amount: 150000,
        created_at: '2024-01-01T00:00:00Z',
      },
    ])

    const branchMock = branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'Main Branch', code: 'MBR' })

    const supplierMock = suppliersApi.getById as unknown as ReturnType<typeof vi.fn>
    supplierMock.mockResolvedValueOnce({ id: 7, code: 'SUP', name: 'Supply Co', is_active: true })

    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <PurchasesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('PO-009')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /complete/i }))
    await user.click(screen.getByRole('button', { name: /confirm/i }))

    await waitFor(() => expect(screen.getByText('COMPLETED')).toBeInTheDocument())
  })

  it('shows API error when the load fails with 403', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = purchasesApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <PurchasesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have access to purchases/i)).toBeInTheDocument())
  })
})
