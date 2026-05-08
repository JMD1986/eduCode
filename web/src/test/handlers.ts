import { http, HttpResponse } from 'msw'

const baseHandlers = [
  http.get('/api/auth/session', () => {
    return HttpResponse.json({ authenticated: false })
  }),
]

export const authHandlers = {
  unauthenticated: [
    http.get('/api/auth/session', () => HttpResponse.json({ authenticated: false })),
  ],
  authenticated: [
    http.get('/api/auth/session', () =>
      HttpResponse.json({ authenticated: true, subject: 'sub:test' }),
    ),
  ],
}

export const handlers = [...baseHandlers]
