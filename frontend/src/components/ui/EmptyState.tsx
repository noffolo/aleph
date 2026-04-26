import React from 'react';

interface EmptyStateProps {
  message?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({ 
  message = "aleph-v2 ❯ _" 
}) => {
  return (
    <div className="flex items-center justify-center py-8 w-full h-full">
      <div className="text-textMuted font-mono text-center text-meta animate-fade-in">
        {message}
      </div>
    </div>
  );
};
