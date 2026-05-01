import React from 'react';
import { useStore } from '../../store/useStore';

interface StatusBarProps {
  projectID: string;
  ollamaHealthy: boolean;
  nlpHealthy: boolean;
}

export const StatusBar: React.FC<StatusBarProps> = React.memo(({ projectID, ollamaHealthy, nlpHealthy }) => {
  const slideOverContent = useStore(s => s.slideOverContent)
  const inlineContent = useStore(s => s.inlineContent)
  const inputMode = useStore(s => s.inputMode)
  const slideOverType = slideOverContent?.type
  const inlineType = inlineContent?.type
  const context = slideOverType
    ? slideOverType.toUpperCase()
    : inlineType
      ? inlineType.toUpperCase()
      : 'READY'

  return (
    <div role="status" aria-live="polite" className="h-7 flex items-center justify-between px-3 py-2 border-t border-border bg-surface font-mono text-[10px] text-textDim shrink-0 select-none leading-snug tracking-widest">
      <div className="flex items-center gap-4">
        <span className="text-primary font-bold terminal-glow">ALEPH</span>
        <span className="text-textDim">│</span>
        <span className="text-textMuted">{projectID || 'NO PROJECT'}</span>
        <span className="text-textDim">│</span>
        <span className="text-textMuted">{context}</span>
        <span className={`ml-2 ${inputMode ? 'text-textMuted' : 'text-primary'} font-bold`}>
          {inputMode ? '[INPUT]' : '[CMD]'}
        </span>
      </div>
      <div className="flex items-center gap-4">
        <span className="flex items-center gap-1">
          <span className={`w-1.5 h-1.5 rounded-full ${ollamaHealthy ? 'bg-success' : 'bg-danger'}`} />
          <span className="text-textMuted">OLLAMA</span>
        </span>
        <span className="flex items-center gap-1">
          <span className={`w-1.5 h-1.5 rounded-full ${nlpHealthy ? 'bg-primary' : 'bg-warning'}`} />
          <span className="text-textMuted">NLP</span>
        </span>
      </div>
    </div>
  );
});