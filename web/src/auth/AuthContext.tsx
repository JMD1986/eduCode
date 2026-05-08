import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { AuthState, SessionResponse } from './types'

type AuthContextValue = AuthState & {
  refreshSession: () => Promise<void>
  signIn: () => void
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

async function fetchSession(): Promise<SessionResponse> {
  const res = await fetch('/api/auth/session', { credentials: 'include' })
  if (!res.ok) return { authenticated: false }
  const body = (await res.json()) as SessionResponse
  return body
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    loading: true,
    authenticated: false,
  })

  const refreshSession = useCallback(async () => {
    try {
      const session = await fetchSession()
      setState({
        loading: false,
        authenticated: session.authenticated,
        subject: session.subject,
      })
    } catch {
      setState({ loading: false, authenticated: false })
    }
  }, [])

  useEffect(() => {
    void refreshSession()
  }, [refreshSession])

  const signIn = useCallback(() => {
    window.location.assign('/api/auth/login')
  }, [])

  const signOut = useCallback(async () => {
    await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    })
    await refreshSession()
  }, [refreshSession])

  const value = useMemo<AuthContextValue>(
    () => ({ ...state, refreshSession, signIn, signOut }),
    [state, refreshSession, signIn, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
