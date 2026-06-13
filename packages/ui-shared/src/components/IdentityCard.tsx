import React from 'react'
import { VerifiedBadge, type AssuranceLevel } from './VerifiedBadge'

export interface IdentityCardProps {
  /** The verified full name (from RealMe Verified Identity) */
  fullName?: string
  /** The verified address line 1 */
  addressLine1?: string
  /** The verified address city */
  city?: string
  /** The verified address postcode */
  postcode?: string
  /** The assurance level of this identity */
  assuranceLevel: AssuranceLevel
  /** Whether to show the address. Defaults to true when address fields are provided. */
  showAddress?: boolean
  /** Called when the user clicks "Change Identity" */
  onChangeIdentity?: () => void
  className?: string
}

/**
 * IdentityCard displays a user's verified identity details in a card format.
 * Used in forms and dashboards to show "who is this form being submitted as".
 *
 * Only shows verified fields when assuranceLevel === 'verified'.
 * For login-level users it shows the badge only.
 */
export function IdentityCard({
  fullName,
  addressLine1,
  city,
  postcode,
  assuranceLevel,
  showAddress = true,
  onChangeIdentity,
  className = '',
}: IdentityCardProps) {
  const hasAddress = showAddress && (addressLine1 || city)

  return (
    <div
      className={`realme-identity-card ${className}`}
      style={{
        border: '1px solid #E5E7EB',
        borderRadius: '8px',
        padding: '16px',
        backgroundColor: '#FFFFFF',
        fontFamily: 'Arial, sans-serif',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
        <span style={{ fontSize: '12px', fontWeight: 600, color: '#6B7280', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Identity
        </span>
        <VerifiedBadge level={assuranceLevel} size="sm" />
      </div>

      {assuranceLevel === 'verified' && fullName && (
        <div style={{ marginBottom: '8px' }}>
          <p style={{ margin: 0, fontSize: '18px', fontWeight: 700, color: '#111827' }}>
            {fullName}
          </p>
        </div>
      )}

      {assuranceLevel === 'verified' && hasAddress && (
        <div style={{ marginTop: '8px', fontSize: '14px', color: '#374151' }}>
          {addressLine1 && <p style={{ margin: '2px 0' }}>{addressLine1}</p>}
          {city && (
            <p style={{ margin: '2px 0' }}>
              {city}
              {postcode ? ` ${postcode}` : ''}
            </p>
          )}
        </div>
      )}

      {assuranceLevel === 'login' && (
        <p style={{ margin: 0, fontSize: '14px', color: '#6B7280' }}>
          Authenticated via RealMe. Verified identity not requested.
        </p>
      )}

      {onChangeIdentity && (
        <button
          type="button"
          onClick={onChangeIdentity}
          style={{
            marginTop: '12px',
            background: 'none',
            border: 'none',
            color: '#1D4ED8',
            fontSize: '13px',
            cursor: 'pointer',
            padding: 0,
            textDecoration: 'underline',
          }}
        >
          Not you? Change identity
        </button>
      )}
    </div>
  )
}
