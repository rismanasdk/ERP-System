import { ApiError } from '../lib/api'
import { readStoredAccessToken, readStoredUser } from '../services/authSession'
import type { User } from '../types/auth'

export function getAccessToken(): string | null {
  return readStoredAccessToken()
}

export function getCurrentUser(): User | null {
  return readStoredUser()
}

export function getApiErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message
  }

  if (error instanceof Error) {
    return error.message
  }

  return 'Something went wrong'
}
