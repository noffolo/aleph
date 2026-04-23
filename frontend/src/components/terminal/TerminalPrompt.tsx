import React, { useRef, useEffect, useState } from 'react';
import { getTabCompletion } from './slashCommands';

interface TerminalPromptProps {
  value: string;
  onChange: (val: string) => void;
  onSubmit: () => void;
  disabled?: boolean;
  placeholder?: string;
  prefix?: string;
}

export const TerminalPrompt: React.FC<TerminalPromptProps> = ({
  value, onChange, onSubmit, disabled = false, placeholder = 'inserisci un comando...', prefix = 'λ'
}) => {
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [tabCompletions, setTabCompletions] = useState<string[]>([]);
  const [completionIndex, setCompletionIndex] = useState(0);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    const completions = getTabCompletion(value);
    setTabCompletions(completions);
    setCompletionIndex(0);
  }, [value]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!disabled && value.trim()) onSubmit();
    } else if (e.key === 'Tab' && tabCompletions.length > 0) {
      e.preventDefault();
      const completion = tabCompletions[completionIndex];
      if (completion) {
        onChange(completion + ' ');
        setTimeout(() => {
          inputRef.current?.setSelectionRange(completion.length + 1, completion.length + 1);
        }, 0);
      }
    }
  };

  return (
    <div className="flex flex-col gap-1 px-4 py-3 border-t border-border bg-surface font-mono text-base leading-relaxed">
      <div className="flex items-start gap-2">
        <span className="text-primary terminal-glow select-none mt-0.5 shrink-0 tracking-tight">{prefix}</span>
        <textarea
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          rows={1}
          className="flex-1 bg-transparent text-text outline-none resize-none terminal-input placeholder:text-textDim text-base leading-relaxed"
          placeholder={placeholder}
        />
      </div>
      
      {tabCompletions.length > 0 && (
        <div className="flex flex-wrap gap-1 ml  
-8 mt-1">
          {tabCompletions.map((cmd, idx) => (
            <span 
              key={cmd} 
              className={`text-xs px-
2 py-0.5 rounded cursor-pointer ${idx === completionIndex ? 'bg-primary text-background' : 'bg-surface-alt text-text'}`}
              onClick={() => { onChange(cmd + ' '); }}
            >
              {cmd}
            </span>
          ))}
        </div>
      )}
    </div>
  );
};