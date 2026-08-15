import type { LoginResponse, RefreshResponse, User } from '../types/auth'

export const STORAGE_KEYS = {
  accessToken: 'erp_access_token',
  refreshToken: 'erp_refresh_token',
  user: 'erp_user',
} as const

export function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEYS.user)
    if (!raw) return null
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function readStoredAccessToken(): string | null {
  return localStorage.getItem(STORAGE_KEYS.accessToken)
}

export function readStoredRefreshToken(): string | null {
  return localStorage.getItem(STORAGE_KEYS.refreshToken)
}

export function persistSession(payload: LoginResponse | RefreshResponse, user: User) {
  localStorage.setItem(STORAGE_KEYS.accessToken, payload.access_token)
  localStorage.setItem(STORAGE_KEYS.refreshToken, payload.refresh_token)
  localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(user))
}

export function clearSession() {
  localStorage.removeItem(STORAGE_KEYS.accessToken)
  localStorage.removeItem(STORAGE_KEYS.refreshToken)
  localStorage.removeItem(STORAGE_KEYS.user)
}
