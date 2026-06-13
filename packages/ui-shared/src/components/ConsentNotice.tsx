import React, { useState } from 'react'

export interface ConsentNoticeProps {
  /** The name of the service requesting consent (e.g. "Business Incorporation Service") */
  serviceName: string
  /** The specific personal information that will be collected from RealMe */
  dataRequested: string[]
  /** The purpose for collecting this information */
  purpose: string
  /** Link to the privacy policy */
  privacyPolicyUrl?: string
  /** Called when the user accepts the consent notice */
  onAccept: () => void
  /** Called when the user declines */
  onDecline?: () => void
  className?: string
}

/**
 * ConsentNotice displays a Privacy Act 2020 compliant consent disclosure
 * before redirecting the user to RealMe for identity verification.
 *
 * Under Privacy Act 2020 Principle 3, agencies must inform individuals of:
 * - Who is collecting the information
 * - Why it is being collected
 * - The consequences of not providing it
 * - Any right of access and correction
 */
export function ConsentNotice({
  serviceName,
  dataRequested,
  purpose,
  privacyPolicyUrl,
  onAccept,
  onDecline,
  className = '',
}: ConsentNoticeProps) {
  const [acknowledged, setAcknowledged] = useState(false)

  return (
    <div
      className={`realme-consent-notice ${className}`}
      role="dialog"
      aria-labelledby="consent-title"
      aria-describedby="consent-body"
      style={{
        border: '2px solid #1D4ED8',
        borderRadius: '8px',
        padding: '24px',
        backgroundColor: '#EFF6FF',
        fontFamily: 'Arial, sans-serif',
        maxWidth: '560px',
      }}
    >
      <h2
        id="consent-title"
        style={{ margin: '0 0 16px', fontSize: '18px', fontWeight: 700, color: '#1E3A5F' }}
      >
        Identity Verification Required
      </h2>

      <div id="consent-body">
        <p style={{ margin: '0 0 12px', fontSize: '14px', color: '#374151', lineHeight: '1.6' }}>
          <strong>{serviceName}</strong> uses RealMe to verify your identity. The following
          information will be provided by the Department of Internal Affairs (DIA):
        </p>

        <ul style={{ margin: '0 0 16px', paddingLeft: '20px', fontSize: '14px', color: '#374151', lineHeight: '1.8' }}>
          {dataRequested.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>

        <p style={{ margin: '0 0 16px', fontSize: '14px', color: '#374151', lineHeight: '1.6' }}>
          <strong>Purpose:</strong> {purpose}
        </p>

        <p style={{ margin: '0 0 16px', fontSize: '12px', color: '#6B7280', lineHeight: '1.6' }}>
          Your information is collected under the Privacy Act 2020. You have the right to
          access and correct information held about you.
          {privacyPolicyUrl && (
            <>
              {' '}
              See our{' '}
              <a href={privacyPolicyUrl} style={{ color: '#1D4ED8' }}>
                Privacy Policy
              </a>
              .
            </>
          )}
        </p>

        <label
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '10px',
            marginBottom: '20px',
            fontSize: '14px',
            color: '#111827',
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            style={{ marginTop: '2px', width: '16px', height: '16px', flexShrink: 0 }}
          />
          I understand how my information will be used and consent to identity verification
          via RealMe.
        </label>
      </div>

      <div style={{ display: 'flex', gap: '12px' }}>
        <button
          type="button"
          onClick={onAccept}
          disabled={!acknowledged}
          style={{
            padding: '10px 24px',
            backgroundColor: acknowledged ? '#00538C' : '#9CA3AF',
            color: '#FFFFFF',
            border: 'none',
            borderRadius: '4px',
            fontSize: '14px',
            fontWeight: 600,
            cursor: acknowledged ? 'pointer' : 'not-allowed',
            flex: 1,
          }}
          aria-disabled={!acknowledged}
        >
          Continue to RealMe
        </button>

        {onDecline && (
          <button
            type="button"
            onClick={onDecline}
            style={{
              padding: '10px 24px',
              backgroundColor: 'transparent',
              color: '#374151',
              border: '1px solid #D1D5DB',
              borderRadius: '4px',
              fontSize: '14px',
              cursor: 'pointer',
            }}
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  )
}
