export type User = {
  id: number
  email: string
  name?: string
  roles?: string[]
  permissions?: string[]
  created_at?: string
  updated_at?: string
}

export type Role = {
  id?: number
  name?: string
}

export type Permission = {
  id?: number
  name?: string
}

export type Branch = {
  id: number
  name: string
  code: string
  is_active?: boolean
  created_at?: string
  updated_at?: string
}

export type LoginRequest = {
  email: string
  password: string
}

export type LoginResponse = {
  access_token: string
  refresh_token: string
  user: User
}

export type RefreshResponse = {
  access_token: string
  refresh_token: string
}

export type APIError = {
  code: string
  message: string
}

export type ApiEnvelope<T> = {
  data: T
  message: string
}

export type ApiErrorEnvelope = {
  error: APIError
}
