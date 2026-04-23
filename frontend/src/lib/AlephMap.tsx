import React from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import L from 'leaflet';

// Fix for default marker icon in Leaflet + React
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

let DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41]
});
L.Marker.prototype.options.icon = DefaultIcon;

interface Row {
  values: { [key: string]: string };
}

interface AlephMapProps {
  rows: Row[];
  onRowClick?: (row: Row) => void;
}

export const AlephMap: React.FC<AlephMapProps> = ({ rows, onRowClick }) => {
  const points = rows.map(r => {
    const lat = parseFloat(r.values['lat'] || r.values['latitude'] || '0');
    const lon = parseFloat(r.values['lon'] || r.values['lng'] || r.values['longitude'] || '0');
    return { lat, lon, data: r };
  }).filter(p => p.lat !== 0 && p.lon !== 0);

  if (points.length === 0) {
    return (
      <div className="h-[500px] flex items-center justify-center bg-surface-alt rounded-lg border-2 border-dashed border-border">
        <p className="text-textMuted font-bold uppercase tracking-widest text-xs">No Geographic Data found in result set</p>
      </div>
    );
  }

  const center: [number, number] = [points[0].lat, points[0].lon];

  return (
    <div className="h-[500px] w-full rounded-lg overflow-hidden shadow-lg border border-border">
      <MapContainer center={center} zoom={10} scrollWheelZoom={false} style={{ height: '100%', width: '100%' }}>
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {points.map((p, i) => (
          <Marker key={i} position={[p.lat, p.lon]} eventHandlers={{ click: () => onRowClick?.(p.data) }}>
            <Popup>
               <div className="p-2">
                  <div className="font-bold border-b mb-2 pb-1">Entity Details</div>
                  {Object.entries(p.data.values).slice(0, 5).map(([k, v]) => (
                    <div key={k} className="text-[10px]"><span className="font-bold uppercase text-textMuted">{k}:</span> {v}</div>
                  ))}
               </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  );
};
