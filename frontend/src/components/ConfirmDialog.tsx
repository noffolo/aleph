import { useEffect, useRef } from 'react';
import { t } from '../i18n';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({ isOpen, title, message, onConfirm, onCancel }: ConfirmDialogProps) {
  const confirmRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    const previousFocus = document.activeElement as HTMLElement;

    setTimeout(() => {
      confirmRef.current?.focus();
    }, 50);

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      if (previousFocus) {
        previousFocus.focus();
      }
    };
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onCancel}
      />

      <div
        className="relative bg-surface border border-border rounded-xl shadow-2xl w-full max-w-sm mx-4 p-6 animate-in fade-in zoom-in duration-200"
        style={{ animation: 'confirm-dialog-in 0.2s cubic-bezier(0.16, 1, 0.3, 1)' }}
      >
        <h3 className="text-lg font-bold text-text mb-2">{title}</h3>
        <p className="text-sm text-textMuted mb-6 leading-relaxed">{message}</p>

        <div className="flex justify-end space-x-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-bold text-textMuted hover:text-text bg-surface-alt hover:bg-border rounded-lg transition-colors"
          >
            {t('confirmDialog.cancel')}
          </button>
          <button
            ref={confirmRef}
            onClick={onConfirm}
            className="px-4 py-2 text-sm font-bold text-white bg-danger hover:bg-danger/90 rounded-lg transition-colors"
          >
            {t('confirmDialog.confirm')}
          </button>
        </div>
      </div>

      <style>{`
        @keyframes confirm-dialog-in {
          from {
            opacity: 0;
            transform: scale(0.95) translateY(8px);
          }
          to {
            opacity: 1;
            transform: scale(1) translateY(0);
          }
        }
      `}</style>
    </div>
  );
}
