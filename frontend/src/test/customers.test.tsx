import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { AuthProvider } from '../contexts/AuthContext'
import { CustomersPage } from '../pages/CustomersPage'
import { ConfirmDialogProvider } from '../utils/confirmUtils'

vi.mock('../services/customers', () => ({
  customersApi: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    softDelete: vi.fn(),
    remove: vi.fn(),
  },
}))

import { customersApi } from '../services/customers'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

function renderCustomersPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <ConfirmDialogProvider>
          <CustomersPage />
        </ConfirmDialogProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('CustomersPage', () => {
  it('renders customer list', async () => {
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([{ id: 1, code: 'C001', name: 'ACME', is_active: true }])

    renderCustomersPage()

    expect(screen.getByPlaceholderText(/search by code or name/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('ACME')).toBeInTheDocument())
  })

  it('shows loading state', () => {
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockReturnValue(new Promise(() => {}))

    renderCustomersPage()

    expect(screen.getByPlaceholderText(/search by code or name/i)).toBeInTheDocument()
  })

  it('shows empty state', async () => {
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText(/no customers found/i)).toBeInTheDocument())
  })

  it('filters by exact search field', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.read'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderCustomersPage()

    const searchInput = await screen.findByPlaceholderText(/search by code or name/i)
    await user.type(searchInput, 'ACME')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith({ search: 'ACME' }, undefined)
    })
  })

  it('filters active=true', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.read'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderCustomersPage()

    const statusSelect = await screen.findByRole('combobox')
    await user.selectOptions(statusSelect, 'true')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith({ active: true }, undefined)
    })
  })

  it('filters active=false', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.read'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderCustomersPage()

    const statusSelect = await screen.findByRole('combobox')
    await user.selectOptions(statusSelect, 'false')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith({ active: false }, undefined)
    })
  })

  it('allows creating a customer successfully', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const createMock = customersApi.create as ReturnType<typeof vi.fn>
    const newCustomer = { id: 2, code: 'C002', name: 'NewCo', is_active: true }
    createMock.mockResolvedValue(newCustomer)

    const user = userEvent.setup()
    renderCustomersPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), newCustomer.code)
    await user.type(screen.getByLabelText(/name/i), newCustomer.name)
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(createMock).toHaveBeenCalledWith(expect.objectContaining({ code: 'C002', name: 'NewCo' }), undefined)
    })
  })

  it('validates required code and name', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderCustomersPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(screen.getByText(/code is required/i)).toBeInTheDocument()
      expect(screen.getByText(/name is required/i)).toBeInTheDocument()
    })
  })

  it('allows updating a customer', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.update'] }))
    const existing = { id: 3, code: 'C003', name: 'OldName', is_active: true }

    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])

    const updateMock = customersApi.update as ReturnType<typeof vi.fn>
    updateMock.mockResolvedValue({ ...existing, name: 'NewName' })

    const user = userEvent.setup()
    renderCustomersPage()

    await waitFor(() => expect(screen.getByText('OldName')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /edit/i }))
    const nameInput = screen.getByLabelText(/name/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'NewName')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(updateMock).toHaveBeenCalledWith(3, expect.objectContaining({ name: 'NewName' }), undefined)
    })
  })

  it('soft-deletes by deactivating an active customer', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.delete'] }))
    const existing = { id: 5, code: 'C005', name: 'ToDelete', is_active: true }

    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])

    const removeMock = customersApi.remove as ReturnType<typeof vi.fn>
    removeMock.mockResolvedValue({ id: existing.id })

    const user = userEvent.setup()
    renderCustomersPage()

    await waitFor(() => expect(screen.getByText('ToDelete')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /deactivate/i }))
    await user.click(screen.getByRole('button', { name: /^Deactivate$/i }))

    await waitFor(() => {
      expect(removeMock).toHaveBeenCalledWith(5, undefined)
    })
  })

  it('hides deactivate button for inactive customer', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.delete'] }))
    const existing = { id: 6, code: 'C006', name: 'InactiveCustomer', is_active: false }

    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText('InactiveCustomer')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /deactivate/i })).not.toBeInTheDocument()
  })

  it('shows session expired state', async () => {
    const err = new ApiError(401, 'Session expired', 'UNAUTHORIZED')
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText(/session expired/i)).toBeInTheDocument())
  })

  it('shows permission denied state', async () => {
    const err = new ApiError(403, 'forbidden', 'FORBIDDEN')
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText(/you do not have access to customers/i)).toBeInTheDocument())
  })

  it('shows duplicate-code API error', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['customers.create'] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const createMock = customersApi.create as ReturnType<typeof vi.fn>
    createMock.mockRejectedValueOnce(new ApiError(400, 'customer code already exists', 'INVALID_REQUEST'))

    const user = userEvent.setup()
    renderCustomersPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), 'DUP')
    await user.type(screen.getByLabelText(/name/i), 'DupCustomer')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => expect(screen.getByText(/customer code already exists/i)).toBeInTheDocument())
  })

  it('shows generic API error', async () => {
    const err = new ApiError(500, 'Server error', 'INTERNAL_SERVER_ERROR')
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockRejectedValueOnce(err)

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText(/server error/i)).toBeInTheDocument())
  })

  it('hides create, edit, and deactivate when permissions are absent', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: [] }))
    const listMock = customersApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([{ id: 1, code: 'C001', name: 'ACME', is_active: true }])

    renderCustomersPage()

    await waitFor(() => expect(screen.getByText('ACME')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /create/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /deactivate/i })).not.toBeInTheDocument()
  })
})
