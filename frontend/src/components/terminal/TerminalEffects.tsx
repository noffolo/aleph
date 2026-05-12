import React, { useEffect, useState } from 'react'
import { useStore } from '../../store/useStore'

export const TerminalEffects: React.FC<{ className?: string }> = ({
  className = '',
}) => {
  const enableScanline = useStore(s => s.enableScanline)
  const enableGlow = useStore(s => s.enableGlow)
  const enableFlicker = useStore(s => s.enableFlicker)
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  useEffect(() => {
    const mql = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPrefersReducedMotion(mql.matches)
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  const reduced = prefersReducedMotion
  const scanlineEnabled = enableScanline && !reduced
  const glowEnabled = enableGlow && !reduced
  const flickerEnabled = enableFlicker && !reduced

  const scanlineOpacity = 0.03
  const glowIntensity = 0.5

  return (
    <div className={`fixed inset-0 pointer-events-none z-[9999] ${className}`}
          aria-hidden="true"
    >
      {/* ── Scanline Overlay ── */}
      {scanlineEnabled && (
        <div
          className="absolute inset-0"
          style={{
            background: `repeating-linear-gradient(
              0deg,
              transparent,
              transparent 2px,
              rgba(0,0,0,${scanlineOpacity}) 2px,
              rgba(0,0,0,${scanlineOpacity}) 4px
            )`,
            backgroundSize: '100% 4px',
            animation: 'scanline-scroll 10s linear infinite',
          }}
        />
      )}

      {/* ── Subtle Horizontal Flicker ── */}
      {flickerEnabled && (
        <div
          className="absolute inset-0"
          style={{
            background: `linear-gradient(
              180deg,
              rgba(0,0,0,0) 0%,
              rgba(0,0,0,0.02) 50%,
              rgba(0,0,0,0) 100%
            )`,
            animation: 'flicker 0.15s infinite',
          }}
        />
      )}

      {/* ── Global Glow Layer ── */}
      {glowEnabled && (
        <div
          className="absolute inset-0"
          style={{
            background: 'radial-gradient(circle at 50% 50%, rgba(0,212,255,0.03) 0%, transparent 70%)',
            mixBlendMode: 'screen',
            opacity: glowIntensity,
          }}
        />
      )}

    </div>
  )
}

export default TerminalEffects
