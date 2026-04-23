import React, { useState } from 'react';

interface Row {
  values: { [key: string]: string };
}

interface AlephTableProps {
  columns: string[];
  rows: Row[];
  onRowClick?: (row: Row) => void;
}

export const AlephTable: React.FC<AlephTableProps> = ({ columns, rows, onRowClick }) => {
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null);

  if (!rows || rows.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 bg-surface border border-border">
        <div className="text-textDim mb-2 font-mono text-xs">Nessun dato trovato</div>
        <p className="text-xs text-textDim text-center font-mono">L'ontologia selezionata non ha restituito risultati dal dataset.</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto border border-border bg-surface">
      <table className="w-full text-left border-collapse font-mono">
        <thead>
          <tr className="bg-surfaceAlt border-b border-border">
            {columns.map((col) => (
              <th key={col} className="px-2 py-1 text-[10px] font-bold text-textMuted uppercase tracking-wider border-b border-border">
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr 
              key={i} 
              onClick={() => {
                setSelectedIdx(i);
                onRowClick?.(row);
              }}
              className={`border-b border-border cursor-pointer transition-colors ${selectedIdx === i ? 'bg-primary/10 text-primary' : 'text-text hover:bg-surfaceAlt'}`}
            >
              {columns.map((col) => (
                <td key={col} className="px-2 py-1 text-xs">
                  <span className="line-clamp-2">{row.values[col] || '—'}</span>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};