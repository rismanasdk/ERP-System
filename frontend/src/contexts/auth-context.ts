import { createContext } from 'react'
import type { User } from '../types/auth'

export type AuthContextValue = {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (payload: { email: string; password: string }) => Promise<void>
  logout: () => void
  refreshSession: () => Promise<boolean>
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)
