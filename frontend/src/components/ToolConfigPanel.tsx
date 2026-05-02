import React, { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { reportError } from '../lib/errorReporter';

interface ToolConfigPanelProps {
  config: Record<string, any>;
  title?: string;
}

export const ToolConfigPanel: React.FC<ToolConfigPanelProps> = ({ config, title = 'Configurazione Strumento' }) => {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(config, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      reportError('ToolConfigPanel', err);
    }
  };

  return (
    <div className="bg-surface p-6 rounded-lg border border-border space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-sm font-bold uppercase tracking-widest text-textMuted">{title}</h3>
        <button 
          onClick={copyToClipboard}
          className="p-2 bg-surface-alt border border-border rounded-lg hover:bg-border transition-colors text-textMuted hover:text-textPrimary"
          title="Copia JSON"
        >
          {copied ? <Check size={14} className="text-success" /> : <Copy size={14} />}
        </button>
      </div>
      <div className="p-4 bg-background border border-border rounded-lg overflow-hidden">
        <pre className="text-xs font-mono text-textPrimary overflow-auto max-h-64">
          {JSON.stringify(config, null, 2)}
        </pre>
      </div>
    </div>
  );
};
