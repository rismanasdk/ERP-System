import { render, screen, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { AppShell } from '../components/layout/AppShell'

afterEach(() => {
  localStorage.clear()
})

function renderShell(userObj?: Record<string, unknown>) {
  if (userObj) localStorage.setItem('erp_user', JSON.stringify(userObj))
  return render(
    <MemoryRouter>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('Sidebar navigation', () => {
  it('shows Users, Roles, Reports when user has those permissions', () => {
    renderShell({ id: 1, permissions: ['users.read', 'roles.read', 'reports.read'] })

    expect(screen.getByText('Users')).toBeInTheDocument()
    expect(screen.getByText('Roles')).toBeInTheDocument()
    expect(screen.getByText('Reports')).toBeInTheDocument()
  })

  it('hides Roles when user lacks roles.read', () => {
    renderShell({ id: 2, permissions: ['users.read', 'reports.read'] })

    expect(screen.queryByText('Roles')).not.toBeInTheDocument()
  })

  it('hides Users when user lacks users.read', () => {
    renderShell({ id: 3, permissions: ['roles.read'] })

    expect(screen.queryByText('Users')).not.toBeInTheDocument()
  })

  it('hides Reports when user lacks reports.read', () => {
    renderShell({ id: 4, permissions: ['users.read'] })

    expect(screen.queryByText('Reports')).not.toBeInTheDocument()
  })

  it('role name does not affect visibility (ADMIN vs WHATEVER)', () => {
    renderShell({ id: 5, roles: ['ADMIN'], permissions: ['roles.read'] })
    expect(screen.getByText('Roles')).toBeInTheDocument()
    // unmount then render another user with different role but same permission
    cleanup()
    renderShell({ id: 6, roles: ['WHATEVER'], permissions: ['roles.read'] })
    expect(screen.getByText('Roles')).toBeInTheDocument()
  })
})
