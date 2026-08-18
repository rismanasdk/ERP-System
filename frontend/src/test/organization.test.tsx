import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { BranchProvider } from '../contexts/BranchContext'
import { OrganizationPage } from '../pages/OrganizationPage'

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

    return new Response(JSON.stringify({ data: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } })
  }))
})

afterEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

function renderOrganizationPage(userObj?: Record<string, unknown>) {
  const currentUser = userObj ?? {
    id: 1,
    email: 'admin@example.com',
    name: 'Risman Hadi Nata',
    roles: ['SUPER_ADMIN'],
  }

  localStorage.setItem('erp_access_token', 'token')
  localStorage.setItem('erp_user', JSON.stringify(currentUser))

  return render(
    <MemoryRouter>
      <AuthProvider>
        <BranchProvider>
          <OrganizationPage />
        </BranchProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('OrganizationPage', () => {
  it('renders heading and branch groups for a super admin', async () => {
    renderOrganizationPage()

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /organization/i })).toBeInTheDocument()
    })

    expect(screen.getAllByText(/super admin/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText('Bandung').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Jakarta').length).toBeGreaterThan(0)
  })

  it('renders access scope for a regular user', async () => {
    renderOrganizationPage({
      id: 7,
      email: 'staff@example.com',
      name: 'Staff User',
      roles: ['ADMIN'],
      permissions: ['inventory.read'],
    })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /organization/i })).toBeInTheDocument()
    })

    expect(screen.getAllByText(/branch access/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText('Bandung').length).toBeGreaterThan(0)
  })
})
