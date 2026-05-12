import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const { mockFuzzySearch } = vi.hoisted(() => {
  const fn = vi.fn()
  fn.mockReturnValue({ score: 1, indices: [] })
  return { mockFuzzySearch: fn }
})

vi.mock('../../utils/fuzzySearch', () => ({
  fuzzySearch: (text: string, query: string) => mockFuzzySearch(text, query),
  HighlightedText: ({ text }: { text: string; indices: number[]; highlightClass: string }) => (
    <span data-testid="highlighted">{text}</span>
  ),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'agents.search': 'Cerca...',
    }
    return map[key] ?? key
  },
}))

import { FuzzySelect } from '../FuzzySelect'

describe('FuzzySelect', () => {
  const options = [
    { value: 'opt1', label: 'Option One' },
    { value: 'opt2', label: 'Option Two' },
    { value: 'opt3', label: 'Another Thing' },
  ]

  beforeEach(() => {
    mockFuzzySearch.mockReset()
    mockFuzzySearch.mockReturnValue({ score: 1, indices: [] })
  })

  it('renders with placeholder when no value selected', () => {
    render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} placeholder="Scegli..." />
    )
    expect(screen.getByText('Scegli...')).toBeInTheDocument()
  })

  it('renders with selected value label', () => {
    render(
      <FuzzySelect value="opt2" options={options} onChange={vi.fn()} />
    )
    expect(screen.getByText('Option Two')).toBeInTheDocument()
  })

  it('opens dropdown on click', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    expect(screen.getByPlaceholderText('Cerca...')).toBeInTheDocument()
  })

  it('shows all options when dropdown opens with no search', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    expect(screen.getByText('Option One')).toBeInTheDocument()
    expect(screen.getByText('Option Two')).toBeInTheDocument()
    expect(screen.getByText('Another Thing')).toBeInTheDocument()
  })

  it('selects an option and calls onChange', () => {
    const onChange = vi.fn()
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={onChange} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    fireEvent.click(screen.getByText('Option Two'))
    expect(onChange).toHaveBeenCalledWith('opt2')
  })

  it('closes dropdown on Escape key', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)

    const searchInput = screen.getByPlaceholderText('Cerca...')
    fireEvent.keyDown(searchInput, { key: 'Escape' })

    expect(screen.queryByPlaceholderText('Cerca...')).not.toBeInTheDocument()
  })

  it('does not open when disabled', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} disabled />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    expect(screen.queryByPlaceholderText('Cerca...')).not.toBeInTheDocument()
  })

  it('displays disabled styling', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} disabled />
    )
    expect(container.querySelector('.opacity-50')).toBeInTheDocument()
  })

  it('shows "Nessun risultato" when no matches', () => {
    mockFuzzySearch.mockReturnValue(null)
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    const searchInput = screen.getByPlaceholderText('Cerca...')
    fireEvent.change(searchInput, { target: { value: 'zzz' } })
    expect(screen.getByText('Nessun risultato')).toBeInTheDocument()
  })

  it('calls fuzzySearch when typing in search input', () => {
    const { container } = render(
      <FuzzySelect value="" options={options} onChange={vi.fn()} />
    )
    const trigger = container.querySelector('.relative > div') as HTMLElement
    fireEvent.click(trigger)
    const searchInput = screen.getByPlaceholderText('Cerca...')
    fireEvent.change(searchInput, { target: { value: 'one' } })
    expect(mockFuzzySearch).toHaveBeenCalled()
  })
})
