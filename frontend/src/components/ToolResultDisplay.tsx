import React from 'react';
import { PieChart } from 'lucide-react';

interface ToolResultDisplayProps {
  result: string | unknown;
}

export const ToolResultDisplay: React.FC<ToolResultDisplayProps> = ({ result }) => {
  if (!result) return <div className="text-textMuted italic text-xs font-mono p-4">Nessun risultato prodotto</div>;

  let parsed: any = result;
  let isJson: boolean;

  if (typeof result === 'string') {
    try {
      parsed = JSON.parse(result);
      isJson = true;
    } catch {
      isJson = false;
    }
  } else {
    isJson = true;
  }

  if (isJson && typeof parsed === 'object' && parsed !== null) {
    if (parsed.error || parsed.status === 'error') {
      return (
        <div className="p-4 bg-danger/10 border border-danger/30 rounded-lg text-danger text-xs font-mono whitespace-pre-wrap break-all">
          <div className="font-bold uppercase tracking-widest mb-2 flex items-center space-x-2">
            <span>Execution Error</span>
          </div>
          {typeof parsed.error === 'string' ? parsed.error : JSON.stringify(parsed.error, null, 2)}
        </div>
      );
    }

    if ('chart_data' in parsed) {
      return (
        <div className="space-y-4">
          <div className="p-4 bg-surface-alt border border-border rounded-lg flex flex-col items-center justify-center text-center space-y-2 min-h-[200px]">
            <PieChart size={40} className="text-primary opacity-50" />
            <div className="text-textPrimary font-bold text-sm uppercase tracking-widest">Visualizzazione Grafica</div>
            <p className="text-textMuted text-xs font-mono">Dati rilevati: {Array.isArray(parsed.chart_data) ? parsed.chart_data.length : 'Oggetto'}</p>
            <div className="text-[10px] text-primary/60 font-bold bg-primary/10 px-2 py-1 rounded">PLACEHOLDER CHART</div>
          </div>
        </div>
      );
    }

    return (
      <div className="overflow-hidden border border-border rounded-lg bg-background">
        <div className="p-4 bg-surface-alt border-b border-border">
          <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">Structured Result</span>
        </div>
        <pre className="p-4 text-xs font-mono text-textPrimary whitespace-pre-wrap overflow-auto max-h-[400px]">
          {JSON.stringify(parsed, null, 2)}
        </pre>
      </div>
    );
  }

  return (
    <pre className="p-4 bg-surface-alt border border-border rounded-lg text-xs font-mono text-textPrimary whitespace-pre-wrap break-all">
      {String(result)}
    </pre>
  );
};
