import React, { useState, useEffect, useRef, useCallback } from 'react';
import { InlineErrorBoundary } from '../InlineErrorBoundary';
import { t } from '../../i18n';

interface SlideOverPanelProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  fullscreen?: boolean;
}

export const SlideOverPanel: React.FC<SlideOverPanelProps> = ({ isOpen, onClose, title, children, fullscreen: initialFullscreen = false }) => {
  const [isFullscreen, setIsFullscreen] = useState(initialFullscreen);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }

      if (e.key === 'Tab') {
        const focusableElements = panelRef.current
          ?.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')
          ;

        const focusableElementsArray = focusableElements ? Array.from(focusableElements) : [];

        if (focusableElementsArray.length === 0) return;

        const firstElement = focusableElementsArray[0] as HTMLElement;
        const lastElement = focusableElementsArray[focusableElementsArray.length - 1] as HTMLElement;

        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            lastElement.focus();
            e.preventDefault();
          }
        } else {
          if (document.activeElement === lastElement) {
            firstElement.focus();
            e.preventDefault();
          }
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    
    const focusable = panelRef.current?.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    if (focusable && focusable.length > 0) {
      (focusable[focusable.length - 1] as HTMLElement).focus();
    }

    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const toggleFullscreen = () => setIsFullscreen(prev => !prev);

  return (
    <div 
      className={`fixed inset-y-0 right-0 z-[90] w-full pointer-events-none transition-all duration-300 ${isFullscreen ? 'max-w-full' : 'max-w-2xl'}`}
      style={{ transitionTimingFunction: 'cubic-bezier(0.16, 1, 0.3, 1)' }}
      aria-hidden="true"
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Slide over panel"
        className="h-full glass-panel border-l border-border shadow-2xl flex flex-col pointer-events-auto animate-slide-in-right"
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
              aria-label={t('slideOver.close')}
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
