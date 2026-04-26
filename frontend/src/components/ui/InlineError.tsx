import React from 'react';
import { X } from 'lucide-react';

interface InlineErrorProps {
  message: string;
  onDismiss?: () => void;
}

export const InlineError: React.FC<InlineErrorProps> = ({ 
  message, 
  onDismiss 
}) => {
  return (
    <div className="border-l-4 border-danger bg-danger/10 p-3 text-danger rounded-r flex items-start gap-3 vol-structural animate-fade-in">
      <div className="flex-1 text-body">
        {message}
      </div>
      {onDismiss && (
        <button 
          onClick={onDismiss}
          className="text-danger hover:text-white transition-colors vol-interactive p-0.5"
          aria-label="Dismiss"
        >
          <X size={14} strokeWidth={2} />
        </button>
      )}
    </div>
  );
};
