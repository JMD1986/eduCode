import type { ReactNode } from 'react'
import { useAuth } from './AuthContext'

export function RequireAuth({ children }: { children: ReactNode }) {
  const auth = useAuth()

  if (auth.loading) {
    return (
      <main className="page" aria-busy="true">
        <p>Checking your session…</p>
      </main>
    )
  }

  if (!auth.authenticated) {
    return (
      <main className="page" aria-labelledby="auth-required-title">
        <h1 id="auth-required-title">Sign in required</h1>
        <p className="page-lede">Please sign in to view your enrollments.</p>
        <button className="btn" type="button" onClick={auth.signIn}>
          Sign in
        </button>
      </main>
    )
  }

  return <>{children}</>
}
