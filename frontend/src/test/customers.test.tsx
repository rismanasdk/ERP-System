import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { AuthProvider } from '../contexts/AuthContext'
import { CustomersPage } from '../pages/CustomersPage'

vi.mock('../services/customers', () => ({
  customersApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    softDelete: vi.fn(),
  },
}))

import { customersApi } from '../services/customers'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('CustomersPage', () => {
  it('renders loading and then list', async () => {
    const fake = [{ id: 1, code: 'C001', name: 'ACME', is_active: true }]
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce(fake)

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByPlaceholderText(/search by code or name/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('ACME')).toBeInTheDocument())
  })

  it('shows empty state', async () => {
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())
  })

  it('shows error when list fails with 403', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have access to customers/i)).toBeInTheDocument())
  })

  it('allows creating a customer successfully', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const newCustomer = { id: 2, code: 'C002', name: 'NewCo', is_active: true }
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([]) // initial
    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())

    const user = userEvent.setup()
    // prepare create + refreshed list
    const createMock = customersApi.create as unknown as ReturnType<typeof vi.fn>
    createMock.mockResolvedValueOnce(newCustomer)
    listMock.mockResolvedValueOnce([newCustomer])

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), newCustomer.code)
    await user.type(screen.getByLabelText(/name/i), newCustomer.name)
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText('NewCo')).toBeInTheDocument())
  })

  it('shows validation errors on empty create form', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])
    const user = userEvent.setup()

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(screen.getByText(/code is required/i)).toBeInTheDocument()
    expect(screen.getByText(/name is required/i)).toBeInTheDocument()
  })

  it('shows API error when create fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([])
    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())

    const user = userEvent.setup()
    const createMock = customersApi.create as unknown as ReturnType<typeof vi.fn>
    const err = new ApiError(400, 'bad request', 'BAD')
    createMock.mockRejectedValueOnce(err)

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), 'X001')
    await user.type(screen.getByLabelText(/name/i), 'FailCo')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText(/bad request/i)).toBeInTheDocument())
  })

  it('allows updating a customer', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.update'] }))
    const existing = { id: 3, code: 'C003', name: 'OldName', is_active: true }
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('OldName')).toBeInTheDocument())
    const user = userEvent.setup()

    const updateMock = customersApi.update as unknown as ReturnType<typeof vi.fn>
    updateMock.mockResolvedValueOnce({ ...existing, name: 'NewName' })
    listMock.mockResolvedValueOnce([{ ...existing, name: 'NewName' }])

    await user.click(screen.getByRole('button', { name: /edit/i }))
    const nameInput = screen.getByLabelText(/name/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'NewName')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText('NewName')).toBeInTheDocument())
  })

  it('shows API error when update fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.update'] }))
    const existing = { id: 4, code: 'C004', name: 'Name4', is_active: true }
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Name4')).toBeInTheDocument())
    const user = userEvent.setup()

    const err = new ApiError(400, 'duplicate code', 'BAD')
    const updateMock = customersApi.update as unknown as ReturnType<typeof vi.fn>
    updateMock.mockRejectedValueOnce(err)

    await user.click(screen.getByRole('button', { name: /edit/i }))
    await user.clear(screen.getByLabelText(/name/i))
    await user.type(screen.getByLabelText(/name/i), 'NewName')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(screen.getByText(/duplicate code/i)).toBeInTheDocument())
  })

  it('allows deleting (soft-delete) a customer', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.delete'] }))
    const existing = { id: 5, code: 'C005', name: 'ToDelete', is_active: true }
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ToDelete')).toBeInTheDocument())

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const deleteMock = customersApi.softDelete as unknown as ReturnType<typeof vi.fn>
    deleteMock.mockResolvedValueOnce({ id: existing.id })
    listMock.mockResolvedValueOnce([])

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /delete/i }))

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())
  })

  it('shows API error when delete fails', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.delete'] }))
    const existing = { id: 6, code: 'C006', name: 'ToDelete2', is_active: true }
    const listMock = customersApi.list as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValueOnce([existing])

    render(
      <MemoryRouter>
        <AuthProvider>
          <CustomersPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ToDelete2')).toBeInTheDocument())

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const err = new ApiError(400, 'cannot delete', 'BAD')
    const deleteMock = customersApi.softDelete as unknown as ReturnType<typeof vi.fn>
    deleteMock.mockRejectedValueOnce(err)

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /delete/i }))

    await waitFor(() => expect(screen.getByText(/cannot delete/i)).toBeInTheDocument())
  })
})
