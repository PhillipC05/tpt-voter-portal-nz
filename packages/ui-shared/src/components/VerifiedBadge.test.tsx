import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'
import { VerifiedBadge } from './VerifiedBadge'

describe('VerifiedBadge', () => {
  it('shows Verified label for verified level', () => {
    render(<VerifiedBadge level="verified" />)
    expect(screen.getByText(/verified/i)).toBeDefined()
  })

  it('shows Login label for login level', () => {
    render(<VerifiedBadge level="login" />)
    expect(screen.getByText(/login/i)).toBeDefined()
  })

  it('shows Unverified label for none level', () => {
    render(<VerifiedBadge level="none" />)
    expect(screen.getByText(/unverified/i)).toBeDefined()
  })

  it('hides label when showLabel=false', () => {
    render(<VerifiedBadge level="verified" showLabel={false} />)
    expect(screen.queryByText(/verified/i)).toBeNull()
  })
})
