import React, { useEffect, useState } from 'react'

interface TerminalEffectsProps {
  enableScanline?: boolean
  enableCRT?: boolean
  enableGlow?: boolean
  scanlineOpacity?: number // 0-1
  glowIntensity?: number // 0-1
  className?: string
}

export const TerminalEffects: React.FC<TerminalEffectsProps> = ({
  enableScanline = true,
  enableCRT = false,
  enableGlow = true,
  scanlineOpacity = 0.03,
  glowIntensity = 0.5,
  className = '',
}) => {
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
  const crtEnabled = enableCRT && !reduced
  const glowEnabled = enableGlow && !reduced

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

      {/* ── CRT Curvature / Vignette ── */}
      {crtEnabled && (
        <div
          className="absolute inset-0"
          style={{
            boxShadow: 'inset 0 0 150px rgba(0,0,0,0.5)',
            borderRadius: '20px',
          }}
        />
      )}

      {/* ── Subtle Horizontal Flicker ── */}
      {scanlineEnabled && (
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

      <style>{`
        @keyframes scanline-scroll {
          0% { background-position: 0 0; }
          100% { background-position: 0 12px; }
        }
        @keyframes flicker {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.97; }
        }
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes slideInRight {
          from { transform: translateX(100%); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
      `}</style>
    </div>
  )
}

export default TerminalEffects
