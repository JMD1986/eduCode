import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from '../App'
import { server } from '../test/server'

afterEach(() => {
  cleanup()
})

function renderWithRoute(initialPath: string) {
  render(
    <MemoryRouter initialEntries={[initialPath]}>
      <App />
    </MemoryRouter>,
  )
}

describe('App routing and auth', () => {
  it('marks the active nav item with aria-current on Home', async () => {
    renderWithRoute('/')
    expect(screen.getByRole('link', { name: /^home$/i })).toHaveAttribute(
      'aria-current',
      'page',
    )
  })

  it('marks the active nav item with aria-current on Classes', () => {
    renderWithRoute('/classes')
    expect(screen.getByRole('link', { name: /^classes$/i })).toHaveAttribute(
      'aria-current',
      'page',
    )
  })

  it('navigates from Home to Classes and renders the Classes screen', async () => {
    const user = userEvent.setup()
    renderWithRoute('/')

    await user.click(screen.getByRole('link', { name: /^classes$/i }))

    expect(
      await screen.findByRole('heading', { level: 1, name: /^classes$/i }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/browse available classes/i),
    ).toBeInTheDocument()
  })

  it('shows guard panel on My classes when unauthenticated', async () => {
    server.use(
      http.get('/api/auth/session', () =>
        HttpResponse.json({ authenticated: false }),
      ),
    )

    renderWithRoute('/my-classes')

    expect(
      await screen.findByRole('heading', { name: /sign in required/i }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('shows Sign out control when authenticated', async () => {
    server.use(
      http.get('/api/auth/session', () =>
        HttpResponse.json({ authenticated: true, subject: 'sub:test' }),
      ),
    )

    renderWithRoute('/')

    expect(await screen.findByRole('button', { name: /sign out/i })).toBeInTheDocument()
  })

  it('clicking Sign in redirects to auth login endpoint', async () => {
    server.use(
      http.get('/api/auth/session', () =>
        HttpResponse.json({ authenticated: false }),
      ),
    )

    const assignSpy = vi
      .spyOn(window.location, 'assign')
      .mockImplementation(() => undefined)

    const user = userEvent.setup()
    renderWithRoute('/')

    await user.click(await screen.findByRole('button', { name: /sign in/i }))

    expect(assignSpy).toHaveBeenCalledWith('/api/auth/login')
    assignSpy.mockRestore()
  })
})
