import React, { useState, useEffect, useRef } from 'react';
import { Command, ArrowRight } from 'lucide-react';
import { fuzzySearch, HighlightedText } from '../utils/fuzzySearch';

interface BaseItem {
  label: string;
  id: string;
  icon?: React.ReactNode;
  colorClass?: string;
}

interface GenericCommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  items: BaseItem[];
  onSelect: (id: string) => void;
  title: string;
  placeholder?: string;
}

type FilteredItem = BaseItem & { score: number; indices: number[] };

export const GenericCommandPalette: React.FC<GenericCommandPaletteProps> = ({
  isOpen, onClose, items, onSelect, title, placeholder = "Cerca..."
}) => {
  const [search, setSearch] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSearch('');
    setSelectedIndex(-1);
  }, [isOpen]);

  if (!isOpen) return null;

  const filteredItems: FilteredItem[] = items
    .map((item: BaseItem) => {
      const res = fuzzySearch(item.label, search);
      return res ? { ...item, ...res } : null;
    })
    .filter((item): item is FilteredItem => item !== null)
    .sort((a, b) => b.score - a.score);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') { onClose(); return; }
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIndex(i => Math.min(i + 1, filteredItems.length - 1)); return; }
    if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIndex(i => Math.max(i - 1, 0)); return; }
    if (e.key === 'Enter') { e.preventDefault(); if (selectedIndex >= 0) onSelect(filteredItems[selectedIndex].id); onClose(); return; }
  };

  useEffect(() => {
    if (listRef.current && selectedIndex >= 0) {
      const el = listRef.current.querySelector(`[data-idx="${selectedIndex}"]`) as HTMLElement;
      el?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-md z-[210] flex items-start justify-center pt-[15vh] p-4 animate-in fade-in duration-200" onClick={onClose} onKeyDown={handleKeyDown}>
      <div 
        className="bg-surface w-full max-w-2xl rounded-lg overflow-hidden border border-border animate-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="p-6 border-b flex items-center space-x-4 bg-surface-alt/50">
          <Command size={24} className="text-primary" />
          <input 
            autoFocus
            value={search}
            onChange={e => { setSearch(e.target.value); setSelectedIndex(0); }}
            placeholder={placeholder}
            className="flex-1 bg-transparent border-none outline-none text-xl font-medium text-text placeholder:text-textDim"
          />
          <div className="px-2 py-1 bg-surface rounded-lg border border-border text-[10px] font-bold text-textMuted">ESC</div>
        </div>

        <div className="max-h-[60vh] overflow-auto p-4 custom-scrollbar" ref={listRef}>
          {search && filteredItems.length > 0 && (
            <div className="mb-6">
              <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest">{title}</div>
              <div className="space-y-1">
                {filteredItems.map((item, idx) => (
                  <button
                    key={item.id}
                    data-idx={idx}
                    onClick={() => { onSelect(item.id); onClose(); }}
                    className={`w-full flex items-center justify-between p-4 rounded-lg transition-colors group ${selectedIndex === idx ? `bg-${item.colorClass || 'primary'}/10` : `hover:bg-${item.colorClass || 'primary'}/10`}`}
                  >
                    <div className={`flex items-center space-x-3 font-bold ${selectedIndex === idx ? `text-${item.colorClass || 'primary'}` : `text-textMuted group-hover:text-${item.colorClass || 'primary'}`}`}>
                      {item.icon && React.cloneElement(item.icon as React.ReactElement, { size: 18 })}
                      <span><HighlightedText text={item.label} indices={item.indices} highlightClass={`text-${item.colorClass || 'primary'}`} /></span>
                    </div>
                    <ArrowRight size={16} className={selectedIndex === idx ? `text-${item.colorClass || 'primary'}/50` : `text-textDim group-hover:text-${item.colorClass || 'primary'}/50`} />
                  </button>
                ))}
              </div>
            </div>
          )}

          {!search && (
            <div className="text-center py-20">
              <Command size={48} className="mx-auto text-textDim mb-4" />
              <p className="text-textMuted font-bold text-sm">Digita per cercare {title.toLowerCase()}...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
