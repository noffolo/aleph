import React, { useEffect } from 'react';
import { t } from '../../i18n';
import { X, AlertCircle } from 'lucide-react';

interface ToastErrorProps {
  message: string;
  onRetry?: () => void;
  onDismiss: () => void;
}

export const ToastError: React.FC<ToastErrorProps> = ({ 
  message, 
  onRetry, 
  onDismiss 
}) => {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss();
    }, 5000);

    return () => clearTimeout(timer);
  }, [onDismiss]);

  return (
    <div aria-live="assertive" aria-atomic="true" className="fixed bottom-4 right-4 z-[300] max-w-sm animate-slide-in-right vol-structural">
      <div role="alert" className="bg-surface border border-danger/30 shadow-lg rounded-lg p-4 flex items-start gap-3">
        <AlertCircle size={18} className="text-danger flex-shrink-0 mt-0.5" />
        
        <div className="flex-1">
          <p className="text-body text-text mb-3">
            {message}
          </p>
          
          <div className="flex justify-end gap-2">
            {onRetry && (
<button 
                 onClick={onRetry}
                 className="text-meta hover:text-primary transition-colors vol-interactive px-2 py-1 rounded bg-surfaceAlt border border-border"
               >
                 {t('toast.retry')}
               </button>
            )}
            <button 
              onClick={onDismiss}
              className="text-textMuted hover:text-text transition-colors vol-interactive p-1"
              aria-label={t('inlineError.dismiss')}
            >
              <X size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
