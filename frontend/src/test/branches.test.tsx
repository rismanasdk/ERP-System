import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
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
    listMock.mockResolvedValue([{ id: 1, name: 'Main Branch', code: 'MBR', is_active: true }])

    renderBranchesPage()

    await waitFor(() => expect(screen.getByText('Main Branch')).toBeInTheDocument())
  })

  it('filters active status using backend-supported active flag only', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['inventory.read'] }))
    const listMock = branchesApi.list as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    const user = userEvent.setup()
    renderBranchesPage()

    const statusSelect = await screen.findByRole('combobox')
    await user.selectOptions(statusSelect, 'false')
    await user.click(screen.getByRole('button', { name: /apply/i }))

    await waitFor(() => {
      expect(listMock).toHaveBeenLastCalledWith(false, undefined)
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
})
