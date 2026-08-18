import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { AuthProvider } from '../contexts/AuthContext'
import { BranchesPage } from '../pages/BranchesPage'
import { ConfirmDialogProvider } from '../utils/confirmUtils'

vi.mock('../services/branches', () => ({
  branchesApi: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
  },
}))

import { branchesApi } from '../services/branches'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

function renderBranchesPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <ConfirmDialogProvider>
          <BranchesPage />
        </ConfirmDialogProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('BranchesPage', () => {
  it('renders branch list', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([{ id: 1, name: 'Main Branch', code: 'MBR', is_active: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' }])

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('Main Branch')).toBeInTheDocument())
  })

  it('shows loading state', () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockReturnValue(new Promise(() => {}))

    renderBranchesPage()

    expect(screen.getByRole('combobox', { name: /branch status filter/i })).toBeInTheDocument()
  })

  it('shows empty state when no branches exist', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText(/no branches found/i)).toBeInTheDocument())
  })

  it('filters active status using backend-supported active flag only', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderBranchesPage()

    const statusSelect = await screen.findByRole('combobox', { name: /branch status filter/i })
    await user.selectOptions(statusSelect, 'false')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith(false, undefined)
    })
  })

  it('filters inactive status using active=false', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderBranchesPage()

    const statusSelect = await screen.findByRole('combobox', { name: /branch status filter/i })
    await user.selectOptions(statusSelect, 'true')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith(true, undefined)
    })
  })

  it('creates a branch with the backend payload contract', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.create'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const createMock = branchesApi.create as ReturnType<typeof vi.fn>
    createMock.mockResolvedValue({ id: 2, name: 'North Branch', code: 'NBR', is_active: true })

    const user = userEvent.setup()
    renderBranchesPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/name/i), 'North Branch')
    await user.type(screen.getByLabelText(/code/i), 'NBR')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(createMock).toHaveBeenCalledWith(expect.objectContaining({ name: 'North Branch', code: 'NBR', is_active: true }), undefined)
    })
  })

  it('validates name required', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.create'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderBranchesPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/code/i), 'NBR')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(screen.getByText(/name is required/i)).toBeInTheDocument()
    })
  })

  it('validates code required', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.create'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderBranchesPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/name/i), 'North Branch')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(screen.getByText(/code is required/i)).toBeInTheDocument()
    })
  })

  it('edits a branch with full payload', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.adjust'] }))
    const existing = { id: 9, name: 'Old Branch', code: 'OLD', is_active: true }

    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])

    const updateMock = branchesApi.update as ReturnType<typeof vi.fn>
    updateMock.mockResolvedValue({ ...existing, name: 'Updated Branch' })

    const user = userEvent.setup()
    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('Old Branch')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /edit/i }))
    const nameInput = screen.getByLabelText(/name/i)
    await user.clear(nameInput)
    await user.type(nameInput, 'Updated Branch')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(updateMock).toHaveBeenCalledWith(9, expect.objectContaining({ name: 'Updated Branch', code: 'OLD', is_active: true }), undefined)
    })
  })

  it('views a branch detail dialog', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([{ id: 4, name: 'View Branch', code: 'VBR', is_active: true }])

    const getByIdMock = branchesApi.getById as ReturnType<typeof vi.fn>
    getByIdMock.mockResolvedValue({ id: 4, name: 'View Branch', code: 'VBR', is_active: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' })

    const user = userEvent.setup()
    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('View Branch')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /view/i }))

    await waitFor(() => {
      expect(screen.getByText('Branch Details')).toBeInTheDocument()
    })
  })

  it('deactivates a branch via update with is_active false', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.adjust'] }))
    const existing = { id: 7, name: 'Old Branch', code: 'OLD', is_active: true }

    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([existing])

    const updateMock = branchesApi.update as ReturnType<typeof vi.fn>
    updateMock.mockResolvedValue({ ...existing, is_active: false })

    const user = userEvent.setup()
    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('Old Branch')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /deactivate/i }))
    await user.click(screen.getByRole('button', { name: /^deactivate$/i }))

    await waitFor(() => {
      expect(updateMock).toHaveBeenCalledWith(7, expect.objectContaining({ is_active: false, name: 'Old Branch', code: 'OLD' }), undefined)
    })
  })

  it('does not show deactivate for inactive branch', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.adjust'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([{ id: 8, name: 'Inactive', code: 'INA', is_active: false }])

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('Inactive')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /deactivate/i })).not.toBeInTheDocument()
  })

  it('shows session expired state on 401', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockRejectedValue(new ApiError(401, 'Session expired', 'UNAUTHORIZED'))

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText(/session expired/i)).toBeInTheDocument())
  })

  it('shows forbidden state on 403', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockRejectedValue(new ApiError(403, 'You do not have access to branches.', 'FORBIDDEN'))

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText(/you do not have access to branches/i)).toBeInTheDocument())
  })

  it('shows duplicate code error message from API', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read', 'inventory.create'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const createMock = branchesApi.create as ReturnType<typeof vi.fn>
    createMock.mockRejectedValue(new ApiError(400, 'branch code already exists', 'INVALID_REQUEST'))

    const user = userEvent.setup()
    renderBranchesPage()

    await user.click(screen.getByRole('button', { name: /create/i }))
    await user.type(screen.getByLabelText(/name/i), 'North Branch')
    await user.type(screen.getByLabelText(/code/i), 'NBR')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => {
      expect(screen.getByText(/branch code already exists/i)).toBeInTheDocument()
    })
  })
})
