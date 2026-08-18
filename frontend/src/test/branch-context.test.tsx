import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useEffect } from 'react'
import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { BranchProvider, useBranch } from '../contexts/BranchContext'
import { useAuth } from '../hooks/useAuth'

function BranchHarness() {
  const { user, logout } = useAuth()
  const { accessibleBranches, selectedBranch, isAllBranches, selectBranch, clearSelection } = useBranch()

  useEffect(() => {
    if (user?.email === 'different@example.com') {
      clearSelection()
    }
  }, [clearSelection, user?.email])

  return (
    <div>
      <div data-testid="user">{user?.email ?? 'no-user'}</div>
      <div data-testid="branch-count">{accessibleBranches.length}</div>
      <div data-testid="selected-branch">{selectedBranch ? selectedBranch.name : 'All Branches'}</div>
      <div data-testid="all-branches">{isAllBranches ? 'all' : 'specific'}</div>
      <button type="button" onClick={() => selectBranch(accessibleBranches[0] ?? null)}>
        pick-first
      </button>
      <button type="button" onClick={() => selectBranch(null)}>
        select-all
      </button>
      <button type="button" onClick={logout}>
        logout
      </button>
    </div>
  )
}

function renderWithBranchProvider(userObj?: Record<string, unknown>) {
  const authUser = userObj ?? { id: 1, email: 'admin@example.com', name: 'Admin', roles: ['SUPER_ADMIN'] }
  localStorage.setItem('erp_access_token', 'token')
  localStorage.setItem('erp_user', JSON.stringify(authUser))

  return render(
    <AuthProvider>
      <BranchProvider>
        <BranchHarness />
      </BranchProvider>
    </AuthProvider>,
  )
}

describe('branch context', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)

      if (url.includes('/api/v1/branches')) {
        return new Response(JSON.stringify({
          data: [
            { id: 1, name: 'Jakarta', code: 'JKT', is_active: true },
            { id: 2, name: 'Bandung', code: 'BDG', is_active: true },
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

  it('loads branches for super admin and exposes all branches option', async () => {
    renderWithBranchProvider()

    await waitFor(() => expect(screen.getByTestId('branch-count')).toHaveTextContent('2'))
    expect(screen.getByTestId('selected-branch')).toHaveTextContent('All Branches')
    expect(screen.getByTestId('all-branches')).toHaveTextContent('all')
  })

  it('restricts non-super-admin to accessible branches without all branches', async () => {
    renderWithBranchProvider({ id: 7, email: 'staff@example.com', name: 'Staff', roles: ['ADMIN'], permissions: ['inventory.read'] })

    await waitFor(() => expect(screen.getByTestId('branch-count')).toHaveTextContent('2'))
    expect(screen.getByTestId('selected-branch')).toHaveTextContent('Jakarta')
    expect(screen.getByTestId('all-branches')).toHaveTextContent('specific')
  })

  it('persists selected branch across reload and clears it on logout', async () => {
    const userObj = userEvent.setup()
    renderWithBranchProvider({ id: 7, email: 'staff@example.com', name: 'Staff', roles: ['ADMIN'], permissions: ['inventory.read'] })

    await waitFor(() => expect(screen.getByTestId('selected-branch')).toHaveTextContent('Jakarta'))

    await userObj.click(screen.getByRole('button', { name: /pick-first/i }))
    await waitFor(() => expect(screen.getByTestId('selected-branch')).toHaveTextContent('Jakarta'))

    const saved = sessionStorage.getItem('erp_selected_branch')
    expect(saved).toContain('Jakarta')

    await userObj.click(screen.getByRole('button', { name: /logout/i }))
    await waitFor(() => expect(sessionStorage.getItem('erp_selected_branch')).toBeNull())
  })
})
