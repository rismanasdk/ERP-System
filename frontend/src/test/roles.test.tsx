import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { RolesPage } from '../pages/RolesPage'
import { ConfirmDialogProvider } from '../utils/confirmUtils'

vi.mock('../services/roles', () => ({
  rolesApi: {
    all: vi.fn(),
    list: vi.fn(),
    get: vi.fn(),
  },
}))

import { rolesApi } from '../services/roles'

afterEach(() => {
  localStorage.clear()
  vi.resetAllMocks()
})

describe('RolesPage', () => {
  it('renders roles list when user has roles.read', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, permissions: ['roles.read'] }))
    const listMock = rolesApi.all as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([
      { id: 1, name: 'ADMIN', description: 'Admin', permissions: ['users.read'] },
    ])

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <RolesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('ADMIN')).toBeInTheDocument())
    expect(listMock).toHaveBeenCalled()
  })

  it('shows permission denied and does not call API when user lacks roles.read', async () => {
    localStorage.setItem('erp_user', JSON.stringify({ id: 2, permissions: ['users.read'] }))
    const listMock = rolesApi.all as unknown as ReturnType<typeof vi.fn>
    listMock.mockResolvedValue([])

    render(
      <MemoryRouter>
        <AuthProvider>
          <ConfirmDialogProvider>
            <RolesPage />
          </ConfirmDialogProvider>
        </AuthProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText(/you do not have permission to view roles/i)).toBeInTheDocument())
    expect(listMock).not.toHaveBeenCalled()
  })
})
