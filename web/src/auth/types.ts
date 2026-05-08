export type SessionResponse = {
  authenticated: boolean
  subject?: string
}

export type AuthState = {
  loading: boolean
  authenticated: boolean
  subject?: string
}
