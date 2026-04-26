import { useEffect, useRef, useCallback } from 'react'
import { X, AlertTriangle, Info, CheckCircle2 } from 'lucide-react'
import { useStore } from '../store/useStore'
import type { ToastMessage } from '../store/uiSlice'

function ToastItem({ toast, onRemove }: { toast: ToastMessage; onRemove: (id: string) => void }) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startTimeRef = useRef<number>(Date.now())
  const elapsedRef = useRef<number>(0)
  const rafRef = useRef<number>(0)
  const barRef = useRef<HTMLDivElement>(null)

  const DURATION = 15000 // 15s

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    if (rafRef.current) {
      cancelAnimationFrame(rafRef.current)
      rafRef.current = 0
    }
  }, [])

  useEffect(() => {
    startTimeRef.current = Date.now()
    elapsedRef.current = 0

    const animate = () => {
      const elapsed = Date.now() - startTimeRef.current + elapsedRef.current
      const pct = Math.min(100, (elapsed / DURATION) * 100)
      if (barRef.current) {
        barRef.current.style.width = `${100 - pct}%`
      }
      if (pct < 100) {
        rafRef.current = requestAnimationFrame(animate)
      }
    }
    rafRef.current = requestAnimationFrame(animate)

    timerRef.current = setTimeout(() => {
      onRemove(toast.id)
    }, DURATION)

    return clearTimer
  }, [toast.id, onRemove, clearTimer])

  const iconMap = {
    error: <AlertTriangle size={16} className="text-danger shrink-0" />,
    info: <Info size={16} className="text-primary shrink-0" />,
    success: <CheckCircle2 size={16} className="text-success shrink-0" />,
  }

  return (
    <div
      className="bg-surface border border-border rounded-lg shadow-xl p-3 w-80 animate-toast-slide-in flex flex-col gap-2"
      style={{ animation: 'toast-slide-in 0.3s cubic-bezier(0.16, 1, 0.3, 1)' }}
    >
      <div className="flex items-start gap-2">
        {iconMap[toast.type]}
        <div className="flex-1 min-w-0">
          <div className="text-xs font-bold text-text uppercase tracking-wider leading-tight">
            {toast.context || (toast.type === 'error' ? 'Errore' : toast.type === 'info' ? 'Info' : 'Completato')}
          </div>
          <div className="text-xs text-textMuted mt-0.5 break-words leading-snug">
            {toast.message}
          </div>
        </div>
        <button
          onClick={() => onRemove(toast.id)}
          className="shrink-0 p-0.5 text-textDim hover:text-text transition-colors"
          aria-label="Chiudi"
        >
          <X size={14} />
        </button>
      </div>

      {toast.retry && toast.type === 'error' && (
        <button
          onClick={() => { onRemove(toast.id); toast.retry?.() }}
          className="self-start px-3 py-1 text-xs font-bold bg-danger/10 text-danger rounded border border-danger/30 hover:bg-danger/20 transition-colors"
        >
          Riprova
        </button>
      )}

      {/* Timer bar */}
      <div className="h-0.5 bg-border rounded-full overflow-hidden">
        <div
          ref={barRef}
          className={`h-full rounded-full transition-none ${
            toast.type === 'error' ? 'bg-danger/50' : toast.type === 'info' ? 'bg-primary/50' : 'bg-success/50'
          }`}
          style={{ width: '100%' }}
        />
      </div>
    </div>
  )
}

export function ToastContainer() {
  const toastMessages = useStore((s) => s.toastMessages)
  const removeToast = useStore((s) => s.removeToast)

  if (!toastMessages || toastMessages.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-[100] flex flex-col-reverse gap-2 pointer-events-none">
      {toastMessages.map((t) => (
        <div key={t.id} className="pointer-events-auto">
          <ToastItem toast={t} onRemove={removeToast} />
        </div>
      ))}
    </div>
  )
}
