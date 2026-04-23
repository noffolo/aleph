import React from 'react';

export const SkeletonLoader = () => (
  <div className="space-y-4 animate-pulse">
    <div className="h-8 bg-surface-alt rounded-md w-3/4"></div>
    <div className="h-12 bg-surface-alt rounded-md w-full"></div>
    <div className="space-y-2">
      <div className="h-6 bg-surface-alt rounded-md w-full"></div>
      <div className="h-6 bg-surface-alt rounded-md w-5/6"></div>
      <div className="h-6 bg-surface-alt rounded-md w-full"></div>
      <div className="h-6 bg-surface-alt rounded-md w-4/6"></div>
    </div>
  </div>
);
