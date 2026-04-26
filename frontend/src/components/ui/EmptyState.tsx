import React, { useState, useEffect } from 'react';

interface EmptyStateProps {
  message?: string;
}

const GHOST_PROMPTS = [
  'aleph-v2 ❯ Prova /explore per esplorare dati',
  'aleph-v2 ❯ Usa /agent per parlare con un agente',
  'aleph-v2 ❯ /help mostra tutti i comandi',
  'aleph-v2 ❯ Cerca /tools per gli strumenti disponibili',
  'aleph-v2 ❯ /predict per le previsioni',
  'aleph-v2 ❯ /library per la libreria asset'
];

export const EmptyState: React.FC<EmptyStateProps> = ({ 
  message = "aleph-v2 ❯ _" 
}) => {
  const [index, setIndex] = useState(0);
  const [displayMessage, setDisplayMessage] = useState(message);
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const interval = setInterval(() => {
      setIsVisible(false);
      
      setTimeout(() => {
        setIndex((prev) => (prev + 1) % GHOST_PROMPTS.length);
        setIsVisible(true);
      }, 500);
    }, 4000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    setDisplayMessage(GHOST_PROMPTS[index]);
  }, [index]);

  return (
    <div className="flex items-center justify-center py-8 w-full h-full">
      <div 
        key={displayMessage}
        className={`text-textMuted font-mono text-center text-meta animate-fade-in transition-opacity duration-500 ${
          isVisible ? 'opacity-100' : 'opacity-0'
        }`}
      >
        {displayMessage}
      </div>
    </div>
  );
};
