import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'slideOver.opened': 'Pannello aperto:',
      'slideOver.title': 'Pannello',
      'slideOver.fullscreen': 'Schermo intero',
      'slideOver.exitFullscreen': 'Esci da schermo intero',
      'slideOver.close': 'Chiudi pannello',
      'slideOver.description': 'Contenuto del pannello:',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = (props: React.SVGProps<SVGSVGElement>) => <svg {...props} data-testid={`icon-${name}`} />
    Comp.displayName = name
    return Comp
  }
  return { Expand: Icon('Expand'), X: Icon('X') }
})

vi.mock('../../InlineErrorBoundary', () => ({
  InlineErrorBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

import { SlideOverPanel } from '../SlideOverPanel'

describe('SlideOverPanel', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    // Setup a mock main-content element that the component tries to set inert on
    const mainContent = document.createElement('div')
    mainContent.id = 'main-content'
    document.body.appendChild(mainContent)
  })

  afterEach(() => {
    const mainContent = document.getElementById('main-content')
    if (mainContent) mainContent.remove()
  })

  it('returns null when isOpen is false', () => {
    const { container } = render(
      <SlideOverPanel isOpen={false} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders dialog with correct role and aria-modal when open', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">Content</SlideOverPanel>
    )
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-label', 'Test Panel')
  })

  it('renders the title in the header', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">Content</SlideOverPanel>
    )
    expect(screen.getByText('Test Panel')).toBeInTheDocument()
  })

  it('renders children content', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Hello World</SlideOverPanel>
    )
    expect(screen.getByText('Hello World')).toBeInTheDocument()
  })

  it('calls onClose when Escape key is pressed', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('calls onClose when backdrop overlay is clicked', () => {
    const { container } = render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    const backdrop = container.querySelector('.fixed.inset-0')
    if (backdrop) fireEvent.click(backdrop)
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('calls onClose when close button is clicked', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    fireEvent.click(screen.getByLabelText('Chiudi pannello'))
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('toggles fullscreen when expand button is clicked', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    const expandBtn = screen.getByLabelText('Schermo intero')
    fireEvent.click(expandBtn)
    // After toggle, label should change to exit fullscreen
    expect(screen.getByLabelText('Esci da schermo intero')).toBeInTheDocument()
  })

  it('shows sr-only live region announcement', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test Panel">Content</SlideOverPanel>
    )
    const announcement = screen.getByText('Pannello aperto: Test Panel')
    expect(announcement).toBeInTheDocument()
    expect(announcement).toHaveAttribute('aria-live', 'polite')
  })

  it('sets inert on main-content when open', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    const mainContent = document.getElementById('main-content')
    expect(mainContent?.hasAttribute('inert')).toBe(true)
  })

  it('uses initial fullscreen state from prop', () => {
    render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test" fullscreen={true}>Content</SlideOverPanel>
    )
    expect(screen.getByLabelText('Esci da schermo intero')).toBeInTheDocument()
  })

  it('applies fullscreen max-w class when fullscreen is toggled', () => {
    const { container } = render(
      <SlideOverPanel isOpen={true} onClose={mockOnClose} title="Test">Content</SlideOverPanel>
    )
    const panelWrapper = container.querySelector('.max-w-2xl')
    expect(panelWrapper).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Schermo intero'))
    const fullscreenWrapper = container.querySelector('.max-w-full')
    expect(fullscreenWrapper).toBeInTheDocument()
  })
})
