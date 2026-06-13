import React from 'react'

export type AssuranceLevel = 'none' | 'login' | 'verified'

export interface VerifiedBadgeProps {
  level: AssuranceLevel
  /** Show full label text. Defaults to true. */
  showLabel?: boolean
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const LEVEL_CONFIG: Record<AssuranceLevel, { label: string; color: string; bg: string; icon: string }> = {
  none: {
    label: 'Unverified',
    color: '#6B7280',
    bg: '#F3F4F6',
    icon: '?',
  },
  login: {
    label: 'RealMe Login',
    color: '#1D4ED8',
    bg: '#DBEAFE',
    icon: '✓',
  },
  verified: {
    label: 'RealMe Verified',
    color: '#065F46',
    bg: '#D1FAE5',
    icon: '✓✓',
  },
}

const SIZE_CONFIG = {
  sm: { padding: '2px 8px', fontSize: '11px', iconSize: '10px' },
  md: { padding: '4px 10px', fontSize: '13px', iconSize: '12px' },
  lg: { padding: '6px 14px', fontSize: '15px', iconSize: '14px' },
}

/**
 * VerifiedBadge displays a user's RealMe identity assurance level as a
 * colour-coded badge. Green = Verified Identity, Blue = Login only, Grey = None.
 */
export function VerifiedBadge({
  level,
  showLabel = true,
  size = 'md',
  className = '',
}: VerifiedBadgeProps) {
  const cfg = LEVEL_CONFIG[level]
  const sz = SIZE_CONFIG[size]

  return (
    <span
      className={`realme-verified-badge ${className}`}
      role="status"
      aria-label={`Identity assurance: ${cfg.label}`}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        backgroundColor: cfg.bg,
        color: cfg.color,
        padding: sz.padding,
        borderRadius: '999px',
        fontSize: sz.fontSize,
        fontWeight: 600,
        fontFamily: 'Arial, sans-serif',
        border: `1px solid ${cfg.color}33`,
        whiteSpace: 'nowrap',
      }}
    >
      <span aria-hidden="true" style={{ fontSize: sz.iconSize }}>
        {cfg.icon}
      </span>
      {showLabel && <span>{cfg.label}</span>}
    </span>
  )
}
