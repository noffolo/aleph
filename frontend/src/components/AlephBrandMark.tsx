import React from 'react';

export const AlephBrandMark: React.FC<{ className?: string; size?: number }> = ({ className = '', size = 32 }) => {
  return (
    <svg 
      width={size} 
      height={size} 
      viewBox="0 0 32 32" 
      fill="none" 
      xmlns="http://www.w3.org/2000/svg"
      className={`${className}`}
    >
      <path 
        d="M16 2C9.34315 2 4 7.34315 4 14C4 16.21 4.83 18.26 6.31 19.84" 
        stroke="currentColor" 
        strokeWidth="2" 
        strokeLinecap="round"
      />
      <path 
        d="M28 16C28 22.66 23.66 28 16 28C13.74 28 11.61 27.31 9.71 26.17" 
        stroke="currentColor" 
        strokeWidth="2" 
        strokeLinecap="round"
      />
      
      <path 
        d="M16 8L22 20H10L16 8Z" 
        fill="currentColor" 
        className="animate-pulse"
      />
      
      <circle cx="16" cy="14" r="1" fill="currentColor" />
      <path 
        d="M22 14H10" 
        stroke="currentColor" 
        strokeWidth="1" 
        strokeDasharray="2 2"
      />
    </svg>
  );
};
