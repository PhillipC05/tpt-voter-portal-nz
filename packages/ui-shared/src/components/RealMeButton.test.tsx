import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'
import { RealMeButton } from './RealMeButton'

describe('RealMeButton', () => {
  it('renders login button by default', () => {
    render(<RealMeButton />)
    const link = screen.getByRole('link')
    expect(link).toBeDefined()
    expect(link.getAttribute('href')).toBe('/auth/realme/login')
    expect(link.getAttribute('aria-label')).toContain('Login with RealMe')
  })

  it('renders verified variant with correct aria-label', () => {
    render(<RealMeButton level="verified" />)
    const link = screen.getByRole('link')
    expect(link.getAttribute('aria-label')).toContain('Verified')
  })

  it('appends return URL as query parameter', () => {
    render(<RealMeButton returnUrl="/dashboard" />)
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toContain('return=')
    expect(link.getAttribute('href')).toContain('%2Fdashboard')
  })

  it('accepts a custom loginUrl', () => {
    render(<RealMeButton loginUrl="/custom/login" />)
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('/custom/login')
  })

  it('applies additional className', () => {
    render(<RealMeButton className="my-class" />)
    const link = screen.getByRole('link')
    expect(link.className).toContain('my-class')
  })
})
