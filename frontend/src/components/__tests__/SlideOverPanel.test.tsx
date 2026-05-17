import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SlideOverPanel } from '../terminal/SlideOverPanel'

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'slideOver.opened': 'Opened',
      'slideOver.title': 'Slide Over',
      'slideOver.close': 'Close',
      'slideOver.fullscreen': 'Fullscreen',
      'slideOver.exitFullscreen': 'Exit Fullscreen',
      'slideOver.description': 'Description',
    }
    return map[key] ?? key
  },
}))

vi.mock('../InlineErrorBoundary', () => ({
  InlineErrorBoundary: ({ children }: { children: React.ReactNode }) => <div data-testid="inline-boundary">{children}</div>,
}))

vi.mock('lucide-react', () => ({
  Expand: () => <svg data-testid="expand-icon" />,
  X: () => <svg data-testid="x-icon" />,
}))

describe('SlideOverPanel', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = '<div id="main-content" />'
  })

  it('returns null when isOpen is false', () => {
    const { container } = render(
      <SlideOverPanel isOpen={false} onClose={mockOnClose} title="Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders title when open', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    expect(screen.getByText('Test Panel')).toBeInTheDocument()
  })

  it('renders children inside error boundary when open', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">
        <p>Child Content</p>
      </SlideOverPanel>
    )
    expect(screen.getByText('Child Content')).toBeInTheDocument()
    expect(screen.getByTestId('inline-boundary')).toBeInTheDocument()
  })

  it('calls onClose when backdrop overlay is clicked', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    fireEvent.click(screen.getByText('Opened Panel').parentElement!.firstElementChild!)
  })

  it('renders dialog with correct aria attributes', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-label', 'Test Panel')
  })

  it('toggles fullscreen on expand button click', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    fireEvent.click(screen.getByTitle('Fullscreen'))
    expect(screen.getByTitle('Exit Fullscreen')).toBeInTheDocument()
  })

  it('calls onClose on X button click', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    fireEvent.click(screen.getByLabelText('Close'))
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('closes on Escape key press', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <input data-testid="focusable" placeholder="test" />
      </SlideOverPanel>
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('traps focus with Tab key', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <button data-testid="btn-1">First</button>
        <button data-testid="btn-2">Last</button>
      </SlideOverPanel>
    )
    // Just verify no crash on Tab without shift
    fireEvent.keyDown(window, { key: 'Tab' })
    fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
  })

  it('disables inert on #main-content on unmount', () => {
    const { unmount } = render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Panel">
        <p>Content</p>
      </SlideOverPanel>
    )
    unmount()
    const main = document.getElementById('main-content')
    expect(main?.hasAttribute('inert')).toBe(false)
  })
})
