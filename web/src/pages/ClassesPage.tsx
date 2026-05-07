import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'

export function ClassesPage() {
  const [classes, setClasses] = useState<Awaited<ReturnType<typeof api.listClasses>> | null>(
    null,
  )
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      setClasses(await api.listClasses())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load classes')
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function onEnroll(classId: string) {
    setError(null)
    setBusyId(classId)
    try {
      await api.enrollInClass(classId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Enrollment failed')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <main className="page" aria-labelledby="classes-title">
      <h1 id="classes-title">Classes</h1>
      <p className="page-lede">
        Browse published classes. Sign in is stubbed for development via{' '}
        <code>X-User-Subject</code> (currently <code>{api.getDevUserSubject()}</code>).
      </p>
      {error ? (
        <p className="banner banner-error" role="alert">
          {error}
        </p>
      ) : null}
      {classes === null ? (
        <p aria-busy="true">Loading catalog…</p>
      ) : classes.length === 0 ? (
        <p>No published classes yet. Add rows in the database or seed data.</p>
      ) : (
        <ul className="class-list">
          {classes.map((c) => (
            <li key={c.id} className="class-card">
              <div className="class-card-head">
                <h2>{c.title}</h2>
                <span className="class-meta">
                  {c.active_enrollments} / {c.capacity} enrolled
                </span>
              </div>
              {c.description ? <p className="class-desc">{c.description}</p> : null}
              <button
                type="button"
                className="btn"
                disabled={busyId === c.id}
                onClick={() => void onEnroll(c.id)}
              >
                {busyId === c.id ? 'Enrolling…' : 'Enroll'}
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
