import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { ConfirmDialogProvider } from '../utils/confirmUtils'
import { ProductsPage } from '../pages/ProductsPage'

vi.mock('../services/products', () => ({
  productsApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    softDelete: vi.fn(),
  },
}))

import { productsApi } from '../services/products'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('ProductsPage', () => {
  it('shows Create button when user has products.create', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['products.create', 'products.read'] }))
    const listMock = productsApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <ProductsPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByRole('button', { name: /create/i })).toBeInTheDocument())
  })

  it('hides Create button when user lacks products.create', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 2, permissions: ['products.read'] }))
    const listMock = productsApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <ProductsPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.queryByRole('button', { name: /create/i })).not.toBeInTheDocument())
  })
})
