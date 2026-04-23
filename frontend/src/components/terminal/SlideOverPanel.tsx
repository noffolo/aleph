import React, { useState } from 'react';
import { InlineErrorBoundary } from '../InlineErrorBoundary';

interface SlideOverPanelProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  fullscreen?: boolean;
}

export const SlideOverPanel: React.FC<SlideOverPanelProps> = ({ isOpen, onClose, title, children, fullscreen: initialFullscreen = false }) => {
  const [isFullscreen, setIsFullscreen] = useState(initialFullscreen);

  if (!isOpen) return null;

  const toggleFullscreen = () => setIsFullscreen(prev => !prev);

  return (
    <div className={`fixed inset-y-0 right-0 z-[90] w-full pointer-events-none transition-all duration-300 ${isFullscreen ? 'max-w-full' : 'max-w-2xl'}`}
      style={{ transitionTimingFunction: 'cubic-bezier(0.16, 1, 0.3, 1)' }}
    >
      <div
        className="h-full bg-surface border-l border-border shadow-2xl flex flex-col pointer-events-auto animate-slide-in-right"
      >
        <div className="h-12 flex items-center justify-between px-5 border-b border-border shrink-0">
          <span className="text-primary text-xs font-bold tracking-widest uppercase">{title}</span>
          <div className="flex items-center space-x-1">
            <button
              onClick={toggleFullscreen}
              className="p-2 hover:bg-surface-alt rounded-lg text-textMuted hover:text-text transition-colors"
              aria-label={isFullscreen ? 'Esci da schermo intero' : 'Schermo intero'}
              title={isFullscreen ? 'Esci da schermo intero' : 'Schermo intero'}
            >
              ⛶
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-surface-alt rounded-lg text-textMuted hover:text-text transition-colors"
              aria-label="Chiudi pannello"
            >
              ✕
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-auto">
          <InlineErrorBoundary label={`slideOver-${title}`}>
            {children}
          </InlineErrorBoundary>
        </div>
      </div>
    </div>
  );
};

export default SlideOverPanel;