'use client'

import React, { useEffect, useRef } from 'react'

export interface QRCodeProps {
  /** The data to encode — typically a verification URL */
  value: string
  /** Size in pixels. Defaults to 200. */
  size?: number
  /** Error correction level. 'H' is highest (recommended for logos). Defaults to 'M'. */
  errorCorrectionLevel?: 'L' | 'M' | 'Q' | 'H'
  /** Optional label shown below the QR code */
  label?: string
  className?: string
}

/**
 * QRCode renders a QR code using the browser canvas API via the qrcode library.
 * Used in the Professional Credentials Wallet for shareable verification links.
 *
 * Example:
 *   <QRCode value="https://credentials.tpt.nz/verify/abc123" label="Scan to verify licence" />
 */
export function QRCode({
  value,
  size = 200,
  errorCorrectionLevel = 'M',
  label,
  className = '',
}: QRCodeProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!canvasRef.current || !value) return

    // Dynamic import to avoid SSR issues
    import('qrcode').then((QRCodeLib) => {
      QRCodeLib.toCanvas(canvasRef.current!, value, {
        width: size,
        margin: 2,
        errorCorrectionLevel,
        color: {
          dark: '#111827',
          light: '#FFFFFF',
        },
      })
    })
  }, [value, size, errorCorrectionLevel])

  return (
    <div
      className={`realme-qrcode ${className}`}
      style={{ display: 'inline-flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}
    >
      <canvas
        ref={canvasRef}
        width={size}
        height={size}
        aria-label={label ?? `QR code for: ${value}`}
        role="img"
        style={{ display: 'block', borderRadius: '4px' }}
      />
      {label && (
        <span
          style={{
            fontSize: '12px',
            color: '#6B7280',
            fontFamily: 'Arial, sans-serif',
            textAlign: 'center',
            maxWidth: `${size}px`,
          }}
        >
          {label}
        </span>
      )}
    </div>
  )
}
