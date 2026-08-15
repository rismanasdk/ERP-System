import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api } from '../lib/api'
import {
  clearSession,
  persistSession,
  readStoredRefreshToken,
  readStoredUser,
  readStoredAccessToken,
} from '../services/authSession'
import type { LoginRequest, User } from '../types/auth'
import { AuthContext, type AuthContextValue } from './auth-context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(readStoredUser)
  const [accessToken, setAccessToken] = useState<string | null>(readStoredAccessToken)
  const [isLoading, setIsLoading] = useState(true)

  const isAuthenticated = Boolean(user && accessToken)

  const refreshSession = useCallback(async () => {
    const refreshToken = readStoredRefreshToken()
    if (!refreshToken) {
      clearSession()
      setUser(null)
      setAccessToken(null)
      return false
    }

    try {
      const response = await api.refresh(refreshToken)
      const currentUser = readStoredUser()
      if (!currentUser) {
        clearSession()
        setUser(null)
        setAccessToken(null)
        return false
      }
      persistSession(response, currentUser)
      setUser(currentUser)
      setAccessToken(response.access_token)
      return true
    } catch {
      clearSession()
      setUser(null)
      setAccessToken(null)
      return false
    }
  }, [])

  useEffect(() => {
    const initializeAuth = async () => {
      const initialUser = readStoredUser()
      const initialToken = readStoredAccessToken()
      if (initialUser && initialToken) {
        setUser(initialUser)
        setAccessToken(initialToken)
      }
      setIsLoading(false)
    }

    void initializeAuth()
  }, [])

  const login = useCallback(async (payload: LoginRequest) => {
    const response = await api.login(payload)
    const nextUser = response.user
    persistSession(response, nextUser)
    setUser(nextUser)
    setAccessToken(response.access_token)
  }, [])

  const logout = useCallback(() => {
    clearSession()
    setUser(null)
    setAccessToken(null)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated,
      isLoading,
      login,
      logout,
      refreshSession,
    }),
    [isAuthenticated, isLoading, login, logout, refreshSession, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}


