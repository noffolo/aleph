import React, { useMemo, useRef, useEffect, useState, useCallback } from 'react'

export type ProgressVariant = 'classic' | 'nested' | 'compact' | 'full'
export type ProgressStatus = 'idle' | 'running' | 'done' | 'error' | 'paused'

interface TerminalProgressBarProps {
  label: string
  percent: number
  current?: number
  total?: number
  elapsedMs?: number
  etaMs?: number
  throughput?: { value: number; unit: string } // e.g. { value: 12.5, unit: 'MB/s' }
  variant?: ProgressVariant
  status?: ProgressStatus
  color?: 'primary' | 'success' | 'warning' | 'danger'
  width?: number // number of block chars
  className?: string
  animate?: boolean
  showSparkline?: boolean
  sparklineHistory?: number[] // last N percentages
}

// ── Unicode Block Architecture ──
// We use a graded block system for sub-character precision (1/8 steps)
const BLOCK = {
  full: '█',                      // U+2588
  partial: ['▏','▎','▍','▌','▋','▊','▉'] as const, // U+258F–U+2597
  empty: '░',                     // U+2591
  trackLeft: '│',                 // U+2502
  trackRight: '│',
  trackTop: '─',
  trackBottom: '─',
  cornerTL: '┌',                  // U+250C
  cornerTR: '┐',                  // U+2510
  cornerBL: '└',                  // U+2514
  cornerBR: '┘',                  // U+2518
}

const SPINNER = ['⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏'] as const
const SPINNER_INTERVAL = 80

const COLOR_MAP: Record<string, string> = {
  primary: 'text-primary',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger',
}

// ── Utilities ──
function formatDuration(ms?: number): string {
  if (ms === undefined || ms < 0) return ''
  if (ms < 1000) return `${Math.floor(ms)}ms`
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  const rem = s % 60
  return `${m}m ${rem}s`
}

function formatETA(ms?: number): string {
  if (!ms || ms < 0 || !isFinite(ms)) return ''
  if (ms < 1000) return '<1s'
  const totalSeconds = Math.ceil(ms / 1000)
  if (totalSeconds < 60) return `${totalSeconds}s`
  const m = Math.floor(totalSeconds / 60)
  const s = totalSeconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function padEnd(str: string, len: number): string {
  return str.length >= len ? str.slice(0, len) : str + ' '.repeat(len - str.length)
}

// ── Sparkline Renderer ──
const TerminalSparkline: React.FC<{ data: number[]; width: number; colorClass: string }> = ({
  data, width, colorClass,
}) => {
  const chars = '▁▂▃▄▅▆▇█' as const
  const values = data.slice(-width)
  if (values.length === 0) return null
  const max = Math.max(...values, 1)
  const min = Math.min(...values)
  const range = max - min || 1

  const line = values.map(v => {
    const idx = Math.floor(((v - min) / range) * (chars.length - 1))
    return chars[Math.min(idx, chars.length - 1)]
  }).join('')

  return <span className={`${colorClass} opacity-60`}>{line}</span>
}

export const TerminalProgressBar: React.FC<TerminalProgressBarProps> = ({
  label,
  percent: rawPercent,
  current,
  total,
  elapsedMs,
  etaMs,
  throughput,
  variant = 'classic',
  status = 'running',
  color = 'primary',
  width = 24,
  className = '',
  animate = true,
  showSparkline = false,
  sparklineHistory = [],
}) => {
  const [spinnerIdx, setSpinnerIdx] = useState(0)
  const [smoothPercent, setSmoothPercent] = useState(0)
  const animFrameRef = useRef<number>()
  const spinnerIntervalRef = useRef<ReturnType<typeof setInterval>>()

  const targetPercent = Math.max(0, Math.min(100, rawPercent))

  // Smooth animation using RAF + ease-out
  useEffect(() => {
    if (!animate) {
      setSmoothPercent(targetPercent)
      return
    }
    const start = smoothPercent
    const diff = targetPercent - start
    const duration = 400
    const startTime = performance.now()

    function easeOutCubic(t: number) {
      const x = 1 - t
      return 1 - x * x * x
    }

    function tick(now: number) {
      const elapsed = now - startTime
      const t = Math.min(elapsed / duration, 1)
      const eased = easeOutCubic(t)
      setSmoothPercent(start + diff * eased)
      if (t < 1) animFrameRef.current = requestAnimationFrame(tick)
    }

    animFrameRef.current = requestAnimationFrame(tick)
    return () => {
      if (animFrameRef.current) cancelAnimationFrame(animFrameRef.current)
    }
  }, [targetPercent, animate])

  // Spinner animation for running state
  useEffect(() => {
    if (status !== 'running') return
    spinnerIntervalRef.current = setInterval(() => {
      setSpinnerIdx(prev => (prev + 1) % SPINNER.length)
    }, SPINNER_INTERVAL)
    return () => {
      if (spinnerIntervalRef.current) clearInterval(spinnerIntervalRef.current)
    }
  }, [status])

  const { barString, filled, partialIdx } = useMemo(() => {
    const totalWidth = Math.max(width, 8)
    const totalBlocks = totalWidth * 8 // 8 sub-steps per char for smoothness
    const filledBlocks = Math.floor((smoothPercent / 100) * totalBlocks)
    const fullChars = Math.floor(filledBlocks / 8)
    const remainder = filledBlocks % 8

    let bar = ''
    for (let i = 0; i < totalWidth; i++) {
      if (i < fullChars) bar += BLOCK.full
      else if (i === fullChars && remainder > 0) bar += BLOCK.partial[remainder - 1]
      else bar += BLOCK.empty
    }
    return { barString: bar, filled: fullChars, partialIdx: remainder }
  }, [smoothPercent, width])

  const spinnerChar = status === 'running' ? SPINNER[spinnerIdx] : status === 'done' ? '●' : status === 'error' ? '✕' : status === 'paused' ? '⏸' : '○'
  const colorClass = COLOR_MAP[color] || COLOR_MAP.primary

  // ── COMPACT variant ──
  if (variant === 'compact') {
    return (
      <span className={`font-mono text-xs whitespace-nowrap ${colorClass} ${className}`}>
        {spinnerChar} {padEnd(label, 14)} {barString} {Math.round(targetPercent)}%
      </span>
    )
  }

  // ── NESTED variant ──
  if (variant === 'nested') {
    return (
      <div className={`font-mono text-xs ${className}`}>
        <div className="flex items-center gap-2">
          <span className={`${colorClass} w-3 text-center`}>{spinnerChar}</span>
          <span className="text-text w-24 truncate">{padEnd(label, 24)}</span>
          <span className={`${colorClass}`}>{barString}</span>
          <span className="text-textMuted w-8 text-right">{Math.round(targetPercent).toString().padStart(3)}%</span>
          {elapsedMs !== undefined && (
            <span className="text-textDim">{formatDuration(elapsedMs)}</span>
          )}
          {etaMs !== undefined && etaMs > 0 && (
            <span className="text-textDim">~{formatETA(etaMs)}</span>
          )}
          {throughput && (
            <span className="text-textMuted">{throughput.value.toFixed(1)} {throughput.unit}</span>
          )}
          {status === 'done' && <span className="text-success ml-1">Completato</span>}
          {status === 'error' && <span className="text-danger ml-1">Fallito</span>}
        </div>
      </div>
    )
  }

  // ── FULL variant (maximally informative) ──
  if (variant === 'full') {
    return (
      <div className={`font-mono text-xs ${className}`}>
        {/* Header line */}
        <div className="flex items-center gap-2 mb-1">
          <span className={`${colorClass} w-3 text-center`}>{spinnerChar}</span>
          <span className="text-text font-medium">{label}</span>
          <span className="text-textDim flex-1 text-right">
            {current !== undefined && total !== undefined && (
              <span>[{current.toLocaleString()} / {total.toLocaleString()}]</span>
            )}
          </span>
        </div>
        {/* Bar line */}
        <div className="flex items-center gap-2">
          <span className="text-textDim">{BLOCK.trackLeft}</span>
          <span className={`${colorClass}`}>{barString}</span>
          <span className="text-textDim">{BLOCK.trackRight}</span>
          <span className="text-textMuted w-8 text-right">{Math.round(targetPercent).toString().padStart(3)}%</span>
        </div>
        {/* Metrics line */}
        <div className="flex items-center gap-3 text-textDim mt-0.5 pl-4">
          {elapsedMs !== undefined && <span>{formatDuration(elapsedMs)} trascorso</span>}
          {etaMs !== undefined && etaMs > 0 && (
            <span>ETA {formatETA(etaMs)}</span>
          )}
          {throughput && (
            <span>{throughput.value.toFixed(1)} {throughput.unit}</span>
          )}
          {showSparkline && sparklineHistory.length > 0 && (
            <TerminalSparkline data={sparklineHistory} width={14} colorClass={colorClass} />
          )}
        </div>
      </div>
    )
  }

  // ── CLASSIC variant (default, balanced) ──
  return (
    <div className={`font-mono text-xs ${className}`}>
      <div className="flex items-center gap-2">
        <span className={`${colorClass} w-3 text-center`}>{spinnerChar}</span>
        <span className="text-text w-24 truncate">{padEnd(label, 24)}</span>
        <span className="text-textDim">{BLOCK.trackLeft}</span>
        <span className={`${colorClass}`}>{barString}</span>
        <span className="text-textDim">{BLOCK.trackRight}</span>
        <span className="text-textMuted w-8 text-right">{Math.round(targetPercent).toString().padStart(3)}%</span>
        {current !== undefined && total !== undefined && (
          <span className="text-textMuted">[{current.toLocaleString()} / {total.toLocaleString()}]</span>
        )}
        {elapsedMs !== undefined && (
          <span className="text-textDim">{formatDuration(elapsedMs)}</span>
        )}
        {etaMs !== undefined && etaMs > 0 && (
          <span className="text-textDim">~{formatETA(etaMs)}</span>
        )}
        {throughput && (
          <span className="text-textMuted">{throughput.value.toFixed(1)} {throughput.unit}</span>
        )}
      </div>
      {showSparkline && sparklineHistory.length > 0 && (
        <div className="pl-[4.5rem] mt-0.5">
          <TerminalSparkline data={sparklineHistory} width={width} colorClass={colorClass} />
        </div>
      )}
    </div>
  )
}

export default TerminalProgressBar
