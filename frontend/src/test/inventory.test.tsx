import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { InventoryPage } from '../pages/InventoryPage'

vi.mock('../services/inventory', () => ({
  inventoryApi: {
    list: vi.fn(),
    create: vi.fn(),
    adjust: vi.fn(),
  },
}))

vi.mock('../services/products', () => ({
  productsApi: {
    getById: vi.fn(),
  },
}))

vi.mock('../services/branches', () => ({
  branchesApi: {
    getById: vi.fn(),
  },
}))

import { inventoryApi } from '../services/inventory'
import { productsApi } from '../services/products'
import { ApiError } from '../lib/api'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('InventoryPage', () => {
  it('renders loading and then list', async () => {
    const fake = [{ id: 1, product_id: 10, branch_id: 2, quantity: 5, created_at: '2020-01-01' }]
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])
    listMock.mockResolvedValueOnce(fake)
    const prodMock = productsApi.getById as unknown as ReturnType<typeof vi.fn>
    prodMock.mockResolvedValueOnce({ id: 10, sku: 'P001', name: 'ProdOne', unit: 'pcs' })
    const branchMock = (await import('../services/branches')).branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'Main', code: 'MAIN' })

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByPlaceholderText(/filter by product id/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('ProdOne')).toBeInTheDocument())
  })

  it('shows empty state', async () => {
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])
    listMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no inventory found/i)).toBeInTheDocument())
  })

  it('shows API error when list fails with 403', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])
    listMock.mockRejectedValueOnce(err)

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have access to inventory/i)).toBeInTheDocument())
  })

  it('allows creating inventory successfully', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.create'] }))
    const newItem = { id: 2, product_id: 20, branch_id: 3, quantity: 10 }
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])
    listMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no inventory found/i)).toBeInTheDocument())

    const user = userEvent.setup()
    const createMock = inventoryApi.create as unknown as ReturnType<typeof vi.fn>
    createMock.mockResolvedValueOnce({ id: newItem.id })
    listMock.mockResolvedValueOnce([newItem])
    const prodMock = productsApi.getById as unknown as ReturnType<typeof vi.fn>
    prodMock.mockResolvedValueOnce({ id: 20, sku: 'P020', name: 'NewProd', unit: 'kg' })
    const branchMock = (await import('../services/branches')).branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 3, name: 'Bandung', code: 'BDG' })

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/product id/i), String(newItem.product_id))
    await user.type(screen.getByLabelText(/branch id/i), String(newItem.branch_id))
    await user.type(screen.getByLabelText(/quantity/i), String(newItem.quantity))
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText('NewProd')).toBeInTheDocument())
  })

  it('allows adjusting inventory', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.adjust'] }))
    const existing = { id: 3, product_id: 30, branch_id: 1, quantity: 2 }
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    // return the existing item for all initial calls
    listMock.mockResolvedValue([existing])
    const prodMock = productsApi.getById as unknown as ReturnType<typeof vi.fn>
    prodMock.mockResolvedValueOnce({ id: 30, sku: 'P030', name: 'AdjProd', unit: 'pcs' })
    const branchMock = (await import('../services/branches')).branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 1, name: 'Pusat', code: 'PST' })

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('AdjProd')).toBeInTheDocument())

    const adjustMock = inventoryApi.adjust as unknown as ReturnType<typeof vi.fn>
    adjustMock.mockResolvedValueOnce({ movement_id: 123 })
    listMock.mockResolvedValueOnce([{ ...existing, quantity: 5 }])

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /adjust/i }))
    await waitFor(() => expect(screen.getByLabelText(/movement type/i)).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText(/movement type/i), 'IN')
    await user.type(screen.getByLabelText(/quantity delta/i), '3')
    await user.click(screen.getByRole('button', { name: /submit/i }))

    await waitFor(() => expect(screen.getByText('5')).toBeInTheDocument())
  })

  it('allows viewing inventory detail', async () => {
    const existing = { id: 4, product_id: 40, branch_id: 2, quantity: 7, created_at: '2023-01-01' }
    const listMock = inventoryApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])
    const prodMock = productsApi.getById as unknown as ReturnType<typeof vi.fn>
    prodMock.mockResolvedValueOnce({ id: 40, sku: 'P040', name: 'ViewProd', unit: 'pcs' })
    const branchMock = (await import('../services/branches')).branchesApi.getById as unknown as ReturnType<typeof vi.fn>
    branchMock.mockResolvedValueOnce({ id: 2, name: 'BranchX', code: 'BX' })

    render(
      <MemoryRouter>
        <AuthProvider>
          <InventoryPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ViewProd')).toBeInTheDocument())
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /view/i }))
    await waitFor(() => expect(screen.getByText(/inventory detail/i)).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    const { getByText: getByTextWithin } = require('@testing-library/dom')
    expect(getByTextWithin(dialog, 'BranchX')).toBeInTheDocument()
    expect(getByTextWithin(dialog, '7')).toBeInTheDocument()
  })
})
