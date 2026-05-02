import React, { useState, useEffect } from 'react'
import { t } from '../i18n'
import { Search, X } from 'lucide-react'

interface ChatSearchBarProps {
  query: string
  setQuery: (q: string) => void
  matchCount: number
}

export const ChatSearchBar: React.FC<ChatSearchBarProps> = ({ query, setQuery, matchCount }) => {
  const [debouncedQuery, setDebouncedQuery] = useState(query)

  useEffect(() => {
    const handler = setTimeout(() => {
      setQuery(debouncedQuery)
    }, 300)
    return () => clearTimeout(handler)
  }, [debouncedQuery, setQuery])

  return (
    <div className="px-4 py-2 border-b border-border bg-surface/50 flex items-center gap-3">
      <div className="relative flex-1">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-textDim" />
        <input
          type="text"
          value={debouncedQuery}
          onChange={(e) => setDebouncedQuery(e.target.value)}
          placeholder={t('copilot.search')}
          className="w-full bg-background border border-border rounded px-8 py-1 text-xs font-mono text-text outline-none focus:border-primary/50 transition-colors"
        />
        {debouncedQuery && (
          <button 
            onClick={() => {
              setDebouncedQuery('')
              setQuery('')
            }} 
            className="absolute right-2 top-1/2 -translate-y-1/2 text-textDim hover:text-text transition-colors"
          >
            <X className="w-3 h-3" />
          </button>
        )}
      </div>
      {matchCount > 0 && (
        <div className="text-[10px] font-mono text-textDim px-2 py-1 bg-background border border-border rounded">
          {matchCount} risultati
        </div>
      )}
    </div>
  )
}
