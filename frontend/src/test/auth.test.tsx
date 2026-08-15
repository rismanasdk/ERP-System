import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { AuthProvider } from '../contexts/AuthContext'
import { useAuth } from '../hooks/useAuth'
import { ProtectedRoute } from '../components/auth/ProtectedRoute'
import { LoginPage } from '../pages/LoginPage'

function TestDashboard() {
  const { user, isAuthenticated, logout } = useAuth()

  return (
    <div>
      <div>{isAuthenticated ? 'authenticated' : 'unauthenticated'}</div>
      <div>{user ? user.email : 'no-user'}</div>
      <button type="button" onClick={logout}>logout</button>
    </div>
  )
}

afterEach(() => {
  localStorage.clear()
})

describe('auth foundation', () => {
  it('renders login page', () => {
    render(
      <MemoryRouter>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  it('redirects unauthenticated users away from protected routes', () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Routes>
            <Route path="/dashboard" element={<ProtectedRoute><TestDashboard /></ProtectedRoute>} />
            <Route path="/login" element={<div>login page</div>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByText('login page')).toBeInTheDocument()
  })

  it('allows authenticated users to access dashboard shell', () => {
    localStorage.setItem('erp_access_token', 'token')
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, email: 'user@example.com', name: 'User' }))

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Routes>
            <Route path="/dashboard" element={<ProtectedRoute><TestDashboard /></ProtectedRoute>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByText('authenticated')).toBeInTheDocument()
    expect(screen.getByText('user@example.com')).toBeInTheDocument()
  })

  it('logout clears authentication state', async () => {
    const user = userEvent.setup()
    localStorage.setItem('erp_access_token', 'token')
    localStorage.setItem('erp_user', JSON.stringify({ id: 1, email: 'user@example.com', name: 'User' }))

    render(
      <MemoryRouter>
        <AuthProvider>
          <TestDashboard />
        </AuthProvider>
      </MemoryRouter>,
    )

    expect(screen.getByText('authenticated')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'logout' }))
    expect(screen.getByText('unauthenticated')).toBeInTheDocument()
  })
})
