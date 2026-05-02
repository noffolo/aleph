import React from 'react';

export function escapeHtml(str: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  };
  return str.replace(/[&<>"']/g, (m) => map[m]);
}

export interface TerminalLine {
  id?: string | number;
  type: 'input' | 'output' | 'error' | 'system' | 'tool';
  content: string;
  timestamp?: number;
}

interface TerminalOutputProps {
  lines: TerminalLine[];
  isStreaming?: boolean;
  onMessageClick?: (id: number) => void;
}

const typeStyles: Record<string, string> = {
  input: 'text-primary',
  output: 'text-text',
  error: 'text-danger',
  system: 'text-textMuted italic',
  tool: 'text-warning',
};

export const TerminalOutput: React.FC<TerminalOutputProps> = ({ lines, isStreaming = false, onMessageClick }) => {
  const [displayContent, setDisplayContent] = React.useState<string>('')
  const streamingRef = React.useRef<string>('')

  React.useEffect(() => {
    if (isStreaming && lines.length > 0) {
      const lastLine = lines[lines.length - 1]
      if (lastLine.type === 'output') {
        streamingRef.current = lastLine.content
        const frame = requestAnimationFrame(() => {
          setDisplayContent(streamingRef.current)
        })
        return () => cancelAnimationFrame(frame)
      }
    } else {
      setDisplayContent('')
    }
  }, [lines, isStreaming])

  return (
    <div className="flex-1 overflow-auto px-4 py-3 font-mono text-sm leading-relaxed custom-scrollbar">
      {lines.map((line, i) => {
        const isLastAndStreaming = isStreaming && i === lines.length - 1 && line.type === 'output'
        return (
          <div 
            key={line.id ?? i} 
            onClick={() => onMessageClick && onMessageClick(i)}
            className={`py-0.5 leading-relaxed cursor-pointer hover:bg-background/50 transition-colors ${typeStyles[line.type] || 'text-text'}`}
          >
            {line.type === 'input' && <span className="text-primary terminal-glow mr-2 text-sm tracking-tight">λ</span>}
            {line.type === 'system' && <span className="text-textDim mr-2 text-[10px] tracking-widest leading-snug">{'→</'}</span>}
            {line.type === 'tool' && <span className="text-warning mr-2 text-xs">⚙</span>}
            <span className="whitespace-pre-wrap text-sm">
              {isLastAndStreaming ? (displayContent || escapeHtml(line.content)) : escapeHtml(line.content)}
            </span>
            {line.timestamp && (
              <span className="text-textDim ml-3 text-[10px] leading-snug">
                {new Date(line.timestamp).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })}
              </span>
            )}
          </div>
        )
      })}
      {isStreaming && (
        <div className="flex items-center gap-1 py-0.5 text-textMuted">
          <span className="terminal-cursor terminal-glow text-primary">█</span>
        </div>
      )}
    </div>
  );
};

