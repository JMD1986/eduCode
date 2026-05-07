import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { HomePage } from './HomePage'

afterEach(() => {
  cleanup()
})

describe('HomePage', () => {
  it('renders the product title', () => {
    render(<HomePage />)
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('NextWork')
  })

  it('explains what learners can do here', () => {
    render(<HomePage />)
    expect(screen.getByText(/browse classes and sign up/i)).toBeInTheDocument()
  })

  it('uses a single main landmark for accessibility', () => {
    render(<HomePage />)
    expect(screen.getAllByRole('main')).toHaveLength(1)
  })
})
