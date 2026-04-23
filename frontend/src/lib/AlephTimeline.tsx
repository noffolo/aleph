import React from 'react';

interface Row {
  values: { [key: string]: string };
}

interface AlephTimelineProps {
  rows: Row[];
  onRowClick?: (row: Row) => void;
}

export const AlephTimeline: React.FC<AlephTimelineProps> = ({ rows, onRowClick }) => {
  const events = rows.map(r => {
    const dateStr = r.values['date'] || r.values['timestamp'] || r.values['created_at'] || '';
    const title = r.values['title'] || r.values['name'] || r.values['oggetto'] || 'Evento';
    const description = r.values['description'] || r.values['descrizione'] || r.values['note'] || r.values['dettaglio'] || Object.entries(r.values).filter(([k]) => !['date','timestamp','created_at','title','name','oggetto'].includes(k)).map(([,v]) => v).filter(Boolean).slice(0, 2).join(' — ') || '';
    return { date: dateStr, title, description, data: r };
  }).filter(e => e.date !== '').sort((a, b) => a.date.localeCompare(b.date));

  if (events.length === 0) {
    return (
      <div className="h-[200px] flex items-center justify-center bg-surface-alt rounded-lg border-2 border-dashed border-border">
        <p className="text-textMuted font-bold uppercase tracking-widest text-xs">No Temporal Data found in result set</p>
      </div>
    );
  }

  return (
    <div className="py-12 px-6">
      <div className="relative border-l-4 border-primary ml-4 space-y-12">
        {events.map((e, i) => (
          <div key={i} className="relative ml-8 group cursor-pointer" onClick={() => onRowClick?.(e.data)}>
            <div className="absolute -left-[42px] top-1 w-6 h-6 bg-surface border-4 border-primary rounded-full group-hover:scale-125 transition-transform"></div>
            <div className="text-xs font-mono font-bold text-primary uppercase tracking-tighter mb-1">{e.date}</div>
            <div className="bg-surface p-5 rounded-lg border border-border shadow-sm group-hover:shadow-lg shadow-primary/5 transition-shadow">
                <h4 className="font-bold text-lg text-text">{e.title}</h4>
                {e.description && <p className="text-sm text-textMuted mt-2 line-clamp-2">{e.description}</p>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
