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
    return { date: dateStr, title, data: r };
  }).filter(e => e.date !== '').sort((a, b) => a.date.localeCompare(b.date));

  if (events.length === 0) {
    return (
      <div className="h-[200px] flex items-center justify-center bg-gray-50 rounded-3xl border-2 border-dashed">
        <p className="text-gray-400 font-bold uppercase tracking-widest text-xs">No Temporal Data found in result set</p>
      </div>
    );
  }

  return (
    <div className="py-12 px-6">
      <div className="relative border-l-4 border-blue-600 ml-4 space-y-12">
        {events.map((e, i) => (
          <div key={i} className="relative ml-8 group cursor-pointer" onClick={() => onRowClick?.(e.data)}>
            <div className="absolute -left-[42px] top-1 w-6 h-6 bg-white border-4 border-blue-600 rounded-full group-hover:scale-125 transition-transform"></div>
            <div className="text-xs font-mono font-bold text-blue-600 uppercase tracking-tighter mb-1">{e.date}</div>
            <div className="bg-white p-5 rounded-2xl border border-gray-100 shadow-sm group-hover:shadow-md transition-shadow">
               <h4 className="font-bold text-lg text-gray-900">{e.title}</h4>
               <p className="text-sm text-gray-500 mt-2 line-clamp-2">Dettaglio asincrono estratto via Aleph Timeline View.</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
