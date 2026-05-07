import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import App from '../App'

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

describe('App routing', () => {
  it('marks the active nav item with aria-current on Home', () => {
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

  it('navigates to My classes', async () => {
    const user = userEvent.setup()
    renderWithRoute('/')

    await user.click(screen.getByRole('link', { name: /my classes/i }))

    expect(
      await screen.findByRole('heading', {
        level: 1,
        name: /my classes/i,
      }),
    ).toBeInTheDocument()
  })
})
