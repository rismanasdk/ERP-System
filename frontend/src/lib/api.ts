import type { ApiEnvelope, ApiErrorEnvelope, LoginRequest, LoginResponse, RefreshResponse, User } from '../types/auth'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

class ApiError extends Error {
  status: number
  code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

function readStoredRefreshToken(): string | null {
  return localStorage.getItem('erp_refresh_token')
}

function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem('erp_user')
    if (!raw) return null
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

function persistSession(payload: LoginResponse | RefreshResponse, user: User) {
  localStorage.setItem('erp_access_token', payload.access_token)
  localStorage.setItem('erp_refresh_token', payload.refresh_token)
  localStorage.setItem('erp_user', JSON.stringify(user))
}

function clearSession() {
  localStorage.removeItem('erp_access_token')
  localStorage.removeItem('erp_refresh_token')
  localStorage.removeItem('erp_user')
}

async function parseResponse<T>(response: Response): Promise<T> {
  const text = await response.text()
  const payload = text ? (JSON.parse(text) as T | ApiErrorEnvelope) : null

  if (!response.ok) {
    const errorPayload = payload as ApiErrorEnvelope | undefined
    const errorMessage = errorPayload?.error?.message ?? 'Request failed'
    const errorCode = errorPayload?.error?.code ?? 'REQUEST_FAILED'
    throw new ApiError(response.status, errorMessage, errorCode)
  }

  return payload as T
}

async function request<T>(path: string, init: RequestInit, token?: string, retryOnUnauthorized = true): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  })

  if (response.status === 401 && retryOnUnauthorized) {
    const refreshToken = readStoredRefreshToken()
    if (refreshToken) {
      try {
        const refreshed = await api.refresh(refreshToken)
        const currentUser = readStoredUser()
        if (currentUser) {
          persistSession(refreshed, currentUser)
          return request<T>(path, init, refreshed.access_token, false)
        }
      } catch {
        clearSession()
      }
    }
  }

  return parseResponse<T>(response)
}

export const api = {
  get: async <T>(path: string, token?: string): Promise<T> => request<T>(path, { method: 'GET' }, token),

  post: async <T>(path: string, body?: unknown, token?: string): Promise<T> =>
    request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined, headers: { 'Content-Type': 'application/json' } }, token),

  put: async <T>(path: string, body?: unknown, token?: string): Promise<T> =>
    request<T>(path, { method: 'PUT', body: body !== undefined ? JSON.stringify(body) : undefined, headers: { 'Content-Type': 'application/json' } }, token),

  delete: async <T>(path: string, token?: string): Promise<T> => request<T>(path, { method: 'DELETE' }, token),

  login: async (payload: LoginRequest): Promise<LoginResponse> => {
    const response = await api.post<ApiEnvelope<LoginResponse>>('/api/v1/auth/login', payload)
    return response.data
  },

  refresh: async (refreshToken: string): Promise<RefreshResponse> => {
    // Call refresh without retryOnUnauthorized to avoid infinite refresh loop when refresh fails
    const res = await request<ApiEnvelope<RefreshResponse>>('/api/v1/auth/refresh', { method: 'POST', body: JSON.stringify({ refresh_token: refreshToken }), headers: { 'Content-Type': 'application/json' } }, undefined, false)
    return res.data
  },

  getDashboardSummary: async <T>(token?: string, branchId?: number): Promise<T> => {
    const query = branchId && branchId > 0 ? `?branch_id=${branchId}` : ''
    const response = await api.get<ApiEnvelope<T>>(`/api/v1/dashboard/summary${query}`, token)
    return response.data
  },
}

export { ApiError }
