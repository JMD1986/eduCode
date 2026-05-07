import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'

export function MyClassesPage() {
  const [classes, setClasses] = useState<Awaited<ReturnType<typeof api.listMyClasses>> | null>(
    null,
  )
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      setClasses(await api.listMyClasses())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load your classes')
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <main className="page" aria-labelledby="my-classes-title">
      <h1 id="my-classes-title">My classes</h1>
      <p className="page-lede">
        Active enrollments for dev user <code>{api.getDevUserSubject()}</code>.
      </p>
      {error ? (
        <p className="banner banner-error" role="alert">
          {error}
        </p>
      ) : null}
      {classes === null ? (
        <p aria-busy="true">Loading…</p>
      ) : classes.length === 0 ? (
        <p>You have no active enrollments.</p>
      ) : (
        <ul className="class-list">
          {classes.map((c) => (
            <li key={c.id} className="class-card">
              <div className="class-card-head">
                <h2>{c.title}</h2>
                <span className="class-meta">
                  enrolled {new Date(c.enrolled_at).toLocaleString()}
                </span>
              </div>
              {c.description ? <p className="class-desc">{c.description}</p> : null}
              <p className="class-footnote">Status: {c.enrollment_status}</p>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
