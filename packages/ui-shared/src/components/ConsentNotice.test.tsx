import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import React from 'react'
import { ConsentNotice } from './ConsentNotice'

const defaultProps = {
  serviceName: 'Test Service',
  dataRequested: ['Full name', 'Date of birth'],
  purpose: 'To verify your identity for test purposes.',
  onAccept: vi.fn(),
}

describe('ConsentNotice', () => {
  it('renders the service name', () => {
    render(<ConsentNotice {...defaultProps} />)
    expect(screen.getByText(/Test Service/)).toBeDefined()
  })

  it('lists all requested data items', () => {
    render(<ConsentNotice {...defaultProps} />)
    expect(screen.getByText(/Full name/)).toBeDefined()
    expect(screen.getByText(/Date of birth/)).toBeDefined()
  })

  it('accept button is disabled until checkbox is checked', () => {
    render(<ConsentNotice {...defaultProps} />)
    const acceptBtn = screen.getByRole('button', { name: /continue|accept/i })
    expect(acceptBtn.hasAttribute('disabled')).toBe(true)
  })

  it('calls onAccept when checkbox checked and button clicked', () => {
    const onAccept = vi.fn()
    render(<ConsentNotice {...defaultProps} onAccept={onAccept} />)

    const checkbox = screen.getByRole('checkbox')
    fireEvent.click(checkbox)

    const acceptBtn = screen.getByRole('button', { name: /continue|accept/i })
    fireEvent.click(acceptBtn)

    expect(onAccept).toHaveBeenCalledOnce()
  })

  it('calls onDecline when decline button clicked', () => {
    const onDecline = vi.fn()
    render(<ConsentNotice {...defaultProps} onDecline={onDecline} />)

    const declineBtn = screen.getByRole('button', { name: /cancel|decline/i })
    fireEvent.click(declineBtn)

    expect(onDecline).toHaveBeenCalledOnce()
  })
})
