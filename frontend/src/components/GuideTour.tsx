import React, { useState } from 'react'
import { X, ChevronLeft, ChevronRight, BookOpen } from 'lucide-react'
import { contextualGuides, type GuideEntry } from '../data/contextualGuides'

const GUIDE_ORDER = [
  'copilot',
  'agents-view',
  'skills-view',
  'tools-view',
  'datasources-view',
  'explore',
  'ontology',
  'library-view',
  'components-view',
  'health',
  'settings',
] satisfies (keyof typeof contextualGuides)[]

interface GuideTourProps {
  onClose: () => void
}

export const GuideTour: React.FC<GuideTourProps> = ({ onClose }) => {
  const [currentIndex, setCurrentIndex] = useState(0)
  const currentKey = GUIDE_ORDER[currentIndex]
  const current: GuideEntry | undefined = contextualGuides[currentKey]
  const total = GUIDE_ORDER.length

  if (!current) return null

  const handlePrev = () => {
    setCurrentIndex(i => Math.max(0, i - 1))
  }

  const handleNext = () => {
    if (currentIndex >= total - 1) {
      onClose()
    } else {
      setCurrentIndex(i => i + 1)
    }
  }

  return (
    <div className="fixed inset-0 z-[400] flex items-end justify-center pointer-events-none">
      <div
        className="absolute inset-0 bg-black/60 pointer-events-auto"
        onClick={onClose}
      />

      <div className="relative z-10 w-full max-w-lg mx-4 mb-8 pointer-events-auto animate-in fade-in slide-in-from-bottom-4 duration-300">
        <div className="bg-surface border border-border rounded-xl shadow-2xl overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-border">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                <BookOpen size={16} className="text-primary" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-text leading-tight">
                  {current.title}
                </h3>
                <span className="text-[10px] text-textDim">
                  {currentIndex + 1} / {total}
                </span>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg hover:bg-surface-alt text-textMuted hover:text-text transition-colors"
              aria-label="Close guide"
            >
              <X size={16} />
            </button>
          </div>

          <div className="px-5 py-4 space-y-4 max-h-[50vh] overflow-y-auto">
            <p className="text-sm text-textMuted leading-relaxed">
              {current.description}
            </p>

            {current.tips.length > 0 && (
              <div className="space-y-1.5">
                <div className="text-[10px] font-bold text-primary uppercase tracking-widest">
                  Suggerimenti
                </div>
                <ul className="space-y-1">
                  {current.tips.map((tip, i) => (
                    <li key={i} className="flex items-start gap-2 text-xs text-textDim">
                      <span className="text-primary mt-0.5 shrink-0">•</span>
                      <span>{tip}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {current.relatedLinks.length > 0 && (
              <div className="space-y-1.5">
                <div className="text-[10px] font-bold text-textDim uppercase tracking-widest">
                  Correlati
                </div>
                <div className="flex flex-wrap gap-2">
                  {current.relatedLinks.map((link, i) => {
                    const isCurrent = link.view === currentKey
                    return (
                      <span
                        key={i}
                        className={`text-[10px] px-2 py-1 rounded font-mono ${
                          isCurrent
                            ? 'bg-primary/10 text-primary'
                            : 'bg-surface-alt text-textMuted'
                        }`}
                      >
                        {link.label}
                      </span>
                    )
                  })}
                </div>
              </div>
            )}
          </div>

          <div className="flex items-center justify-between px-5 py-3 border-t border-border bg-surface-alt/50">
            <button
              onClick={handlePrev}
              disabled={currentIndex === 0}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium text-textMuted hover:text-text hover:bg-surface disabled:opacity-30 disabled:cursor-not-allowed transition-all"
            >
              <ChevronLeft size={14} />
              Precedente
            </button>

            <div className="flex items-center gap-1.5">
              {GUIDE_ORDER.map((_, i) => (
                <button
                  key={i}
                  onClick={() => setCurrentIndex(i)}
                  className={`w-1.5 h-1.5 rounded-full transition-all ${
                    i === currentIndex
                      ? 'bg-primary w-4'
                      : 'bg-border hover:bg-textMuted'
                  }`}
                  aria-label={`Go to guide ${i + 1}`}
                />
              ))}
            </div>

            <button
              onClick={handleNext}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium bg-primary text-white hover:bg-primary/90 transition-all"
            >
              {currentIndex >= total - 1 ? 'Chiudi' : 'Prossimo'}
              {currentIndex < total - 1 && <ChevronRight size={14} />}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
