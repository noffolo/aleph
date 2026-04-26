import React from 'react';

export interface FuzzyResult {
  score: number;
  indices: number[];
}

export function fuzzySearch(text: string, query: string): FuzzyResult | null {
  const t = text.toLowerCase();
  const q = query.toLowerCase();

  if (!q) return { score: 1, indices: [] };
  if (!t) return null;

  let tIdx = 0;
  let qIdx = 0;
  let consecutiveBonus = 0;
  const matchedIndices: number[] = [];

  while (tIdx < t.length && qIdx < q.length) {
    if (t[tIdx] === q[qIdx]) {
      matchedIndices.push(tIdx);
      qIdx++;
    } else {
      consecutiveBonus = 0;
    }
    tIdx++;
  }

  if (qIdx !== q.length) return null;

  let score = 0;
  let lastIdx = -2;

  for (let i = 0; i < matchedIndices.length; i++) {
    const idx = matchedIndices[i];

    // Exact match bonus
    if (i === 0 && idx === 0) score += 20;

    // Word-start (after space/underscore/hyphen) bonus
    if (idx > 0 && /[\s\-_]/.test(t[idx - 1])) score += 15;

    // Consecutive match bonus
    if (idx === lastIdx + 1) {
      consecutiveBonus += 10;
      score += consecutiveBonus;
    } else {
      consecutiveBonus = 0;
    }

    // Base point for every match
    score += 5;
    lastIdx = idx;
  }

  // Penalize position (later matches = slightly less score)
  if (matchedIndices.length > 0) {
    score -= matchedIndices[0] * 0.5;
  }

  // Boost exact substring matches
  if (t.includes(q)) score += matchedIndices.length * 3;

  return { score: Math.round(score * 100) / 100, indices: matchedIndices };
}

/**
 * Highlights matching characters in a string based on fuzzy search indices.
 */
export function HighlightedText({ text, indices, highlightClass }: { text: string, indices: number[], highlightClass: string }) {
  const parts = [];
  let lastIdx = 0;

  indices.forEach((idx, i) => {
    if (idx > lastIdx) {
      parts.push(<span key={`p-${i}`}>{text.slice(lastIdx, idx)}</span>);
    }
    parts.push(<span key={`m-${i}`} className={highlightClass}>{text[idx]}</span>);
    lastIdx = idx + 1;
  });

  if (lastIdx < text.length) {
    parts.push(<span key="p-final">{text.slice(lastIdx)}</span>);
  }

  return <>{parts}</>;
}
