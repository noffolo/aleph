import { describe, it, expect } from 'vitest'
import { cn } from '../utils'

describe('cn (className utility)', () => {
  it('returns merged class string for single input', () => {
    expect(cn('text-red-500')).toBe('text-red-500')
  })

  it('merges multiple class strings', () => {
    expect(cn('text-red-500', 'bg-blue-500')).toContain('text-red-500')
    expect(cn('text-red-500', 'bg-blue-500')).toContain('bg-blue-500')
  })

  it('handles conditional classes with falsy values', () => {
    expect(cn('base', false && 'hidden', null, undefined)).toBe('base')
  })

  it('handles empty input', () => {
    expect(cn()).toBe('')
  })

  it('tailwind-merge resolves conflicts (last wins)', () => {
    const result = cn('px-4', 'px-2')
    // tailwind-merge should keep px-2 and drop px-4
    expect(result).toContain('px-2')
    expect(result).not.toContain('px-4')
  })

  it('handles object syntax via clsx', () => {
    const result = cn('base', { 'text-red-500': true, 'hidden': false })
    expect(result).toContain('base')
    expect(result).toContain('text-red-500')
    expect(result).not.toContain('hidden')
  })
})
