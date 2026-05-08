import type { ApiErrorBody, Class, MyClass } from './types'

async function readJson<T>(res: Response): Promise<T | null> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text) as T
  } catch {
    return null
  }
}

export async function listClasses(): Promise<Class[]> {
  const res = await fetch('/api/classes', { credentials: 'include' })
  if (!res.ok) {
    const body = await readJson<ApiErrorBody>(res)
    throw new Error(body?.error ?? `list classes failed (${res.status})`)
  }
  const data = (await res.json()) as { classes: Class[] }
  return data.classes
}

export async function getClass(id: string): Promise<Class> {
  const res = await fetch(`/api/classes/${encodeURIComponent(id)}`, { credentials: 'include' })
  if (!res.ok) {
    const body = await readJson<ApiErrorBody>(res)
    throw new Error(body?.error ?? `get class failed (${res.status})`)
  }
  const data = (await res.json()) as { class: Class }
  return data.class
}

export async function enrollInClass(classId: string): Promise<void> {
  const res = await fetch(`/api/classes/${encodeURIComponent(classId)}/enroll`, {
    method: 'POST',
    credentials: 'include',
  })
  if (res.status === 204) return
  const body = await readJson<ApiErrorBody>(res)
  throw new Error(body?.error ?? `enroll failed (${res.status})`)
}

export async function listMyClasses(): Promise<MyClass[]> {
  const res = await fetch('/api/me/classes', {
    credentials: 'include',
  })
  if (!res.ok) {
    const body = await readJson<ApiErrorBody>(res)
    throw new Error(body?.error ?? `my classes failed (${res.status})`)
  }
  const data = (await res.json()) as { classes: MyClass[] }
  return data.classes
}
