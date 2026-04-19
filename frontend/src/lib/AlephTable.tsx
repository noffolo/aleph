import React from 'react';

interface Row {
  values: { [key: string]: string };
}

interface AlephTableProps {
  columns: string[];
  rows: Row[];
  onRowClick?: (row: Row) => void;
}

export const AlephTable: React.FC<AlephTableProps> = ({ columns, rows, onRowClick }) => {
  if (!rows || rows.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 bg-gray-50 rounded-2xl border-2 border-dashed border-gray-200">
        <div className="text-gray-400 mb-2">Nessun dato trovato</div>
        <p className="text-sm text-gray-500 text-center">L'ontologia selezionata non ha restituito risultati dal dataset.</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-2xl border border-gray-200 bg-white">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="bg-gray-50 border-b border-gray-200">
            {columns.map((col) => (
              <th key={col} className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {rows.map((row, i) => (
            <tr 
              key={i} 
              onClick={() => onRowClick && onRowClick(row)}
              className="hover:bg-blue-50/50 transition-colors cursor-pointer group"
            >
              {columns.map((col) => (
                <td key={col} className="px-6 py-4 text-sm text-gray-700">
                  <span className="line-clamp-2">{row.values[col] || '-'}</span>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
