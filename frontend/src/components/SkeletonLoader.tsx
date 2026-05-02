import React from 'react'

interface SkeletonLoaderProps {
  rows?: number
  cols?: number
  className?: string
}

export const SkeletonLoader = ({ rows = 1, cols = 1, className = '' }: SkeletonLoaderProps) => {
  return (
    <div className={`flex flex-col gap-3 w-full ${className}`}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex gap-3">
          {Array.from({ length: cols }).map((_, j) => (
            <div 
              key={j} 
              className="h-4 bg-white/5 animate-pulse rounded-sm flex-1" 
            />
          ))}
        </div>
      ))}
    </div>
  )
}

interface SkeletonListProps {
  itemCount?: number
  className?: string
}

export const SkeletonList = ({ itemCount = 5, className = '' }: SkeletonListProps) => {
  return (
    <div className={`flex flex-col gap-2 w-full ${className}`}>
      {Array.from({ length: itemCount }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 p-2 bg-white/5 animate-pulse rounded-md">
          <div className="w-4 h-4 bg-white/10 rounded-full shrink-0" />
          <div className="h-3 bg-white/10 rounded-sm flex-1" />
        </div>
      ))}
    </div>
  )
}

