import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { BranchProvider } from '../contexts/BranchContext'
import { ConfirmDialogProvider } from '../utils/confirmUtils'
import { UsersPage } from '../pages/UsersPage'

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)

    if (url.includes('/api/v1/branches')) {
      return new Response(JSON.stringify({
        data: [
          { id: 1, name: 'Bandung', code: 'BDG', is_active: true },
          { id: 2, name: 'Jakarta', code: 'JKT', is_active: true },
        ],
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    if (url.includes('/api/v1/auth/refresh')) {
      return new Response(JSON.stringify({ data: { access_token: 'token', refresh_token: 'refresh' } }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    if (url.includes('/api/v1/users')) {
      return new Response(JSON.stringify({ data: [
        { id: 10, name: 'Budi', email: 'budi@example.com', roles: ['MANAGER'], permissions: ['users.read'], branches: [{ id: 1, name: 'Bandung' }], is_active: true },
        { id: 11, name: 'Ani', email: 'ani@example.com', roles: ['CASHIER'], permissions: [], branches: [{ id: 2, name: 'Jakarta' }], is_active: true },
      ] }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }

    return new Response(JSON.stringify({ data: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } })
  }))
})

afterEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

function renderUsersPage(userObj?: Record<string, unknown>) {
  const currentUser = userObj ?? {
    id: 1,
    email: 'admin@example.com',
    name: 'Risman Hadi Nata',
    roles: ['SUPER_ADMIN'],
    permissions: ['users.read'],
  }

  localStorage.setItem('erp_access_token', 'token')
  localStorage.setItem('erp_user', JSON.stringify(currentUser))

  return render(
    <MemoryRouter>
      <AuthProvider>
        <BranchProvider>
          <ConfirmDialogProvider>
            <UsersPage />
          </ConfirmDialogProvider>
        </BranchProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('UsersPage', () => {
  it('renders user list for an authorized user', async () => {
    renderUsersPage()

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /user management/i })).toBeInTheDocument()
    })

    expect(screen.getByText('Budi')).toBeInTheDocument()
    expect(screen.getByText('Ani')).toBeInTheDocument()
  })

  it('blocks access when the user lacks users.read permission', async () => {
    renderUsersPage({
      id: 7,
      email: 'staff@example.com',
      name: 'Staff User',
      roles: ['ADMIN'],
      permissions: ['inventory.read'],
    })

    await waitFor(() => {
      expect(screen.getByText(/you do not have permission to view user management/i)).toBeInTheDocument()
    })
  })
})
