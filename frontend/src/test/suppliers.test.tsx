import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { SuppliersPage } from '../pages/SuppliersPage'

vi.mock('../services/suppliers', () => ({
  suppliersApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    softDelete: vi.fn(),
  },
}))

import { suppliersApi } from '../services/suppliers'
import { ApiError } from '../lib/api'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('SuppliersPage', () => {
  it('renders loading and then list', async () => {
    const fake = [{ id: 1, code: 'S001', name: 'SupplyCo', is_active: true }]
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce(fake)

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByPlaceholderText(/search by code or name/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('SupplyCo')).toBeInTheDocument())
  })

  it('shows empty state', async () => {
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no suppliers found/i)).toBeInTheDocument())
  })

  it('shows error when list fails with 403', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have access to suppliers/i)).toBeInTheDocument())
  })

  it('allows creating a supplier successfully', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.create'] }))
    const newSupplier = { id: 2, code: 'S002', name: 'NewSupply', is_active: true }
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([]) // initial
    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no suppliers found/i)).toBeInTheDocument())

    const user = userEvent.setup()
    const createMock = suppliersApi.create as unknown as ReturnType<typeof vi.fn>
    createMock.mockResolvedValueOnce(newSupplier)
    listMock.mockResolvedValueOnce([newSupplier])

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), newSupplier.code)
    await user.type(screen.getByLabelText(/name/i), newSupplier.name)
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText('NewSupply')).toBeInTheDocument())
  })

  it('shows validation errors on empty create form', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.create'] }))
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])
    const user = userEvent.setup()

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no suppliers found/i)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(screen.getByText(/code is required/i)).toBeInTheDocument()
    expect(screen.getByText(/name is required/i)).toBeInTheDocument()
  })

  it('shows API error when create fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.create'] }))
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])
    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no suppliers found/i)).toBeInTheDocument())

    const user = userEvent.setup()
    const createMock = suppliersApi.create as unknown as ReturnType<typeof vi.fn>
    const err = new ApiError(400, 'bad request', 'BAD')
    createMock.mockRejectedValueOnce(err)

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), 'X001')
    await user.type(screen.getByLabelText(/name/i), 'FailSupply')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText(/bad request/i)).toBeInTheDocument())
  })

  it('allows updating a supplier', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.update'] }))
    const existing = { id: 3, code: 'S003', name: 'OldSupply', is_active: true }
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('OldSupply')).toBeInTheDocument())
    const user = userEvent.setup()

    const updateMock = suppliersApi.update as unknown as ReturnType<typeof vi.fn>
    updateMock.mockResolvedValueOnce({ ...existing, name: 'NewSupply' })
    listMock.mockResolvedValueOnce([{ ...existing, name: 'NewSupply' }])

    await user.click(screen.getByRole('button', { name: /edit/i }))
    const nameInput = screen.getByLabelText(/name/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'NewSupply')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText('NewSupply')).toBeInTheDocument())
  })

  it('shows API error when update fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.update'] }))
    const existing = { id: 4, code: 'S004', name: 'Supply4', is_active: true }
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Supply4')).toBeInTheDocument())
    const user = userEvent.setup()

    const err = new ApiError(400, 'duplicate code', 'BAD')
    const updateMock = suppliersApi.update as unknown as ReturnType<typeof vi.fn>
    updateMock.mockRejectedValueOnce(err)

    await user.click(screen.getByRole('button', { name: /edit/i }))
    await user.clear(screen.getByLabelText(/name/i))
    await user.type(screen.getByLabelText(/name/i), 'NewSupply')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText(/duplicate code/i)).toBeInTheDocument())
  })

  it('allows deleting (soft-delete) a supplier', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.delete'] }))
    const existing = { id: 5, code: 'S005', name: 'ToDelete', is_active: true }
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ToDelete')).toBeInTheDocument())

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const deleteMock = suppliersApi.softDelete as unknown as ReturnType<typeof vi.fn>
    deleteMock.mockResolvedValueOnce({ id: existing.id })
    listMock.mockResolvedValueOnce([])

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /delete/i }))

    await waitFor(() => expect(screen.getByText(/no suppliers found/i)).toBeInTheDocument())
  })

  it('shows API error when delete fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['suppliers.delete'] }))
    const existing = { id: 6, code: 'S006', name: 'ToDelete2', is_active: true }
    const listMock = suppliersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <SuppliersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ToDelete2')).toBeInTheDocument())

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const err = new ApiError(400, 'cannot delete', 'BAD')
    const deleteMock = suppliersApi.softDelete as unknown as ReturnType<typeof vi.fn>
    deleteMock.mockRejectedValueOnce(err)

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /delete/i }))

    await waitFor(() => expect(screen.getByText(/cannot delete/i)).toBeInTheDocument())
  })
})
