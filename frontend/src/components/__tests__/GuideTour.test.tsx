import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { GuideTour } from '../GuideTour'

vi.mock('../../data/contextualGuides', () => ({
  contextualGuides: {
    copilot: { title: 'Guide 0', description: 'Description 0', tips: ['Tip 1'], relatedLinks: [] },
    'agents-view': { title: 'Guide 1', description: 'Description 1', tips: [], relatedLinks: [] },
    'skills-view': { title: 'Guide 2', description: 'Description 2', tips: [], relatedLinks: [] },
    'tools-view': { title: 'Guide 3', description: 'Description 3', tips: [], relatedLinks: [] },
    'datasources-view': { title: 'Guide 4', description: 'Description 4', tips: [], relatedLinks: [] },
    explore: { title: 'Guide 5', description: 'Description 5', tips: [], relatedLinks: [] },
    ontology: { title: 'Guide 6', description: 'Description 6', tips: [], relatedLinks: [] },
    'library-view': { title: 'Guide 7', description: 'Description 7', tips: [], relatedLinks: [] },
    'components-view': { title: 'Guide 8', description: 'Description 8', tips: [], relatedLinks: [] },
    health: { title: 'Guide 9', description: 'Description 9', tips: [], relatedLinks: [] },
    settings: { title: 'Guide 10', description: 'Description 10', tips: [], relatedLinks: [] },
  },
}))

vi.mock('lucide-react', () => ({
  X: () => null,
  ChevronLeft: () => null,
  ChevronRight: () => null,
  BookOpen: () => null,
}))

describe('GuideTour', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the first guide title', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('Guide 0')).toBeInTheDocument()
  })

  it('renders guide description', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('Description 0')).toBeInTheDocument()
  })

  it('renders step counter', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('1 / 11')).toBeInTheDocument()
  })

  it('renders prev and next buttons', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('Precedente')).toBeInTheDocument()
    expect(screen.getByText('Prossimo')).toBeInTheDocument()
  })

  it('prev button is disabled on first step', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('Precedente')).toBeDisabled()
  })

  it('advances to next step on next click', () => {
    render(<GuideTour onClose={mockOnClose} />)
    fireEvent.click(screen.getByText('Prossimo'))
    expect(screen.getByText('Guide 1')).toBeInTheDocument()
    expect(screen.getByText('2 / 11')).toBeInTheDocument()
  })

  it('closes guide on last step next click', () => {
    render(<GuideTour onClose={mockOnClose} />)
    for (let i = 0; i < 10; i++) {
      fireEvent.click(screen.getByText('Prossimo'))
    }
    expect(screen.getByText('Chiudi')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Chiudi'))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('closes guide on X button click', () => {
    render(<GuideTour onClose={mockOnClose} />)
    fireEvent.click(screen.getByLabelText('Close guide'))
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('renders tips when available', () => {
    render(<GuideTour onClose={mockOnClose} />)
    expect(screen.getByText('Suggerimenti')).toBeInTheDocument()
    expect(screen.getByText('Tip 1')).toBeInTheDocument()
  })
})
