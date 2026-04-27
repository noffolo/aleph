import React, { useEffect } from 'react'
import { t } from '../i18n'
import { useStore } from '../store/useStore'
import { X } from 'lucide-react'

export const ToastBar: React.FC = () => {
  const errorToast = useStore((state) => state.errorToast)
  const clearErrorToast = useStore((state) => state.clearErrorToast)

  useEffect(() => {
    if (errorToast) {
      const timer = setTimeout(() => {
        clearErrorToast()
      }, 5000)
      return () => clearTimeout(timer)
    }
  }, [errorToast, clearErrorToast])

  if (!errorToast) return null

  const colors: Record<string, string> = {
    error: 'bg-red-500/90 border-red-600 text-red-50',
    success: 'bg-green-500/90 border-green-600 text-green-50',
    info: 'bg-blue-500/90 border-blue-600 text-blue-50',
  }

  const colorClass = colors[errorToast.type] || colors.info

  return (
    <div className="fixed bottom-4 right-4 z-50 animate-in slide-in-from-right-full fade-in duration-300">
      <div className={`flex items-center gap-3 px-4 py-3 rounded-lg border ${colorClass} shadow-lg backdrop-blur-md min-w-[300px]`}>
        <div className="flex-1 text-sm font-medium">
          {errorToast.message}
        </div>
        <button
          onClick={clearErrorToast}
          className="p-1 hover:bg-white/20 rounded-full transition-colors"
          aria-label={t('toast.closeNotification')}
        >
          <X size={16} />
        </button>
      </div>
    </div>
  )
}
