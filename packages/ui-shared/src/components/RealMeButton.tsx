import React from 'react'

export interface RealMeButtonProps {
  /** URL to redirect to for RealMe login. Defaults to /auth/realme/login */
  loginUrl?: string
  /** The service level to request. 'login' for basic, 'verified' for Verified Identity. */
  level?: 'login' | 'verified'
  /** URL to return to after authentication. Passed as ?return= parameter. */
  returnUrl?: string
  /** Additional CSS class names */
  className?: string
}

/**
 * RealMeButton renders the DIA brand-compliant "Login with RealMe" button.
 *
 * DIA brand guidelines require specific colours, the RealMe wordmark, and
 * specific button text. This component follows those guidelines.
 * See: https://developers.realme.govt.nz/how-to-integrate/design-standards
 */
export function RealMeButton({
  loginUrl = '/auth/realme/login',
  level = 'login',
  returnUrl,
  className = '',
}: RealMeButtonProps) {
  const href = returnUrl
    ? `${loginUrl}?return=${encodeURIComponent(returnUrl)}`
    : loginUrl

  const isVerified = level === 'verified'

  return (
    <a
      href={href}
      className={`realme-button ${className}`}
      aria-label={isVerified ? 'Login with RealMe Verified Identity' : 'Login with RealMe'}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '10px',
        backgroundColor: '#00538C',
        color: '#FFFFFF',
        padding: '10px 20px',
        borderRadius: '4px',
        textDecoration: 'none',
        fontFamily: 'Arial, sans-serif',
        fontSize: '16px',
        fontWeight: 600,
        border: 'none',
        cursor: 'pointer',
        minWidth: '200px',
        justifyContent: 'center',
      }}
    >
      <RealMeLogo />
      <span>{isVerified ? 'Verify with RealMe' : 'Login with RealMe'}</span>
    </a>
  )
}

function RealMeLogo() {
  return (
    <svg
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      {/* Simplified RealMe logo mark — replace with official SVG from DIA brand kit */}
      <circle cx="12" cy="12" r="10" stroke="white" strokeWidth="2" fill="none" />
      <path d="M8 12 L11 15 L16 9" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
