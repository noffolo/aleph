import { useState, useEffect } from 'react';

export type AlephState = 'IDLE' | 'STREAMING' | 'ERROR' | 'RECOVERING';

export function useAlephStream(dataStream: any) {
  const [state, setState] = useState<AlephState>('IDLE');
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    if (!dataStream) {
      setState('IDLE');
      return;
    }

    setState('STREAMING');
    
    // Logica di ricezione stream
    const handleStream = async () => {
      try {
        for await (const chunk of dataStream) {
          setData(chunk);
          setState('STREAMING');
        }
      } catch (e) {
        console.error("Stream error:", e);
        setState('ERROR');
        // Tentativo di recupero automatico dopo 3 secondi
        setTimeout(() => setState('RECOVERING'), 3000);
      }
    };

    handleStream();
  }, [dataStream]);

  return { state, data };
}
