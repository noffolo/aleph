import React, { useState, useRef, useEffect } from 'react';
import { t } from '../i18n';
import { fuzzySearch, HighlightedText } from '../utils/fuzzySearch';

interface FuzzySelectProps {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

export const FuzzySelect: React.FC<FuzzySelectProps> = ({
  value,
  options,
  onChange,
  placeholder = 'Scegli...',
  disabled = false,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selectedLabel = options.find((o) => o.value === value)?.label ?? value;

  useEffect(() => {
    if (isOpen) {
      setSearch('');
      setSelectedIndex(0);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [isOpen]);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [isOpen]);

  const filtered =
    search.trim() === ''
      ? options.map((o) => ({ ...o, score: 1, indices: [] as number[] }))
      : options
          .map((o) => {
            const res = fuzzySearch(o.label, search);
            return res ? { ...o, ...res } : null;
          })
          .filter(
            (
              item
            ): item is {
              value: string;
              label: string;
              score: number;
              indices: number[];
            } => item !== null
          )
          .sort((a, b) => b.score - a.score);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      setIsOpen(false);
      return;
    }
    if (!isOpen) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1));
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      const item = filtered[selectedIndex];
      if (item) {
        onChange(item.value);
        setIsOpen(false);
      }
      return;
    }
  };

  return (
    <div ref={containerRef} className="relative w-full">
      <div
        onClick={() => {
          if (!disabled) setIsOpen((v) => !v);
        }}
        className={`w-full p-3 bg-background rounded-lg border border-border text-sm flex items-center justify-between cursor-pointer select-none transition-colors ${
          disabled ? 'opacity-50 cursor-not-allowed' : 'hover:border-primary/50'
        }`}
      >
        <span className={`truncate ${value ? 'text-text' : 'text-textDim'}`}>
          {selectedLabel || placeholder}
        </span>
        <svg
          className={`w-4 h-4 text-textDim transition-transform duration-150 ${
            isOpen ? 'rotate-180' : ''
          }`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </div>

      {isOpen && (
        <div className="absolute z-[110] w-full mt-1 bg-surface border border-border rounded-lg shadow-elevation2 overflow-hidden animate-in fade-in zoom-in-95 duration-200">
          <div className="p-2 border-b border-border">
            <input
              ref={inputRef}
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setSelectedIndex(0);
              }}
              onKeyDown={handleKeyDown}
              placeholder={t('agents.search')}
              className="w-full p-2 bg-background rounded-md border border-border text-sm text-text placeholder:text-textDim outline-none focus:border-primary/50"
            />
          </div>

          <div className="max-h-52 overflow-auto custom-scrollbar">
            {filtered.length === 0 ? (
              <div className="p-3 text-sm text-textDim text-center">Nessun risultato</div>
            ) : (
              filtered.map((item, idx) => (
                <button
                  key={item.value}
                  onClick={() => {
                    onChange(item.value);
                    setIsOpen(false);
                  }}
                  className={`w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2 ${
                    selectedIndex === idx
                      ? 'bg-primary/10 text-primary'
                      : 'text-text hover:bg-surface-alt'
                  }`}
                  onMouseEnter={() => setSelectedIndex(idx)}
                >
                  {item.value === value && (
                    <span className="text-primary text-xs">●</span>
                  )}
                  {item.value !== value && <span className="text-transparent text-xs">●</span>}
                  <span className="flex-1 truncate">
                    <HighlightedText
                      text={item.label}
                      indices={item.indices}
                      highlightClass="text-primary font-bold"
                    />
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};
