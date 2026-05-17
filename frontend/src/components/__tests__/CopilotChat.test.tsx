import React from 'react'
import { render, screen, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CopilotChat } from '../CopilotChat'
import type { TerminalLine } from '../terminal/TerminalOutput'

vi.mock('../terminal', () => ({
  TerminalOutput: ({ lines, isStreaming }: { lines: TerminalLine[]; isStreaming: boolean }) => (
    <div data-testid="terminal-output">
      {lines.map((l, i) => <div key={i} data-testid={`line-${l.id}`}>{l.content}</div>)}
      {isStreaming && <div data-testid="streaming-indicator">streaming</div>}
    </div>
  ),
}))

class MockIntersectionObserver {
  root: Element | Document | null = null
  rootMargin = ''
  thresholds: ReadonlyArray<number> = []
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
  takeRecords = vi.fn(() => [])
  static instances: MockIntersectionObserver[] = []
  constructor(_callback: IntersectionObserverCallback, _options?: IntersectionObserverInit) {
    MockIntersectionObserver.instances.push(this)
  }
}

describe('CopilotChat', () => {
  let lastObserver: MockIntersectionObserver

  beforeAll(() => {
    if (!Element.prototype.scrollTo) {
      Element.prototype.scrollTo = vi.fn() as unknown as typeof Element.prototype.scrollTo
    }
  })

  beforeEach(() => {
    vi.clearAllMocks()
    MockIntersectionObserver.instances = []
    window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver
  })

  afterEach(() => {
    lastObserver = MockIntersectionObserver.instances[MockIntersectionObserver.instances.length - 1]
  })

  it('renders TerminalOutput component', () => {
    render(<CopilotChat lines={[]} isStreaming={false} />)
    expect(screen.getByTestId('terminal-output')).toBeDefined()
  })

  it('passes lines to TerminalOutput', () => {
    const lines: TerminalLine[] = [
      { id: 1, content: 'Hello', type: 'output' },
      { id: 2, content: 'World', type: 'output' },
    ]
    render(<CopilotChat lines={lines} isStreaming={false} />)
    expect(screen.getByTestId('line-1')).toBeDefined()
    expect(screen.getByTestId('line-2')).toBeDefined()
  })

  it('passes isStreaming to TerminalOutput', () => {
    render(<CopilotChat lines={[]} isStreaming={true} />)
    expect(screen.getByTestId('streaming-indicator')).toBeDefined()
  })

  it('creates IntersectionObserver on mount', () => {
    render(<CopilotChat lines={[]} isStreaming={false} />)
    expect(MockIntersectionObserver.instances.length).toBeGreaterThan(0)
    expect(MockIntersectionObserver.instances[0].observe).toHaveBeenCalled()
  })

  it('disconnects IntersectionObserver on unmount', () => {
    const { unmount } = render(<CopilotChat lines={[]} isStreaming={false} />)
    const obs = MockIntersectionObserver.instances[0]
    unmount()
    expect(obs.disconnect).toHaveBeenCalled()
  })

  it('renders sentinel div for scroll detection', () => {
    const { container } = render(<CopilotChat lines={[]} isStreaming={false} />)
    const sentinel = container.querySelector('.h-px')
    expect(sentinel).toBeDefined()
  })

  it('renders scroll-to-bottom button when lines present', () => {
    const { container } = render(<CopilotChat lines={[{ id: 1, content: 'msg', type: 'output' }]} isStreaming={false} />)
    const btn = container.querySelector('button[aria-label="Scolla verso il basso"]')
    expect(btn).toBeDefined()
  })

  it('scroll button not shown when isAtBottom is true (default)', () => {
    const { container } = render(<CopilotChat lines={[]} isStreaming={false} />)
    const btn = container.querySelector('button[aria-label="Scolla verso il basso"]')
    expect(btn).toBeNull()
  })

  function setupObserver(signalIntersecting: boolean) {
    window.IntersectionObserver = class {
      root = null; rootMargin = ''; thresholds = []
      observe = vi.fn(); disconnect = vi.fn(); unobserve = vi.fn()
      takeRecords = vi.fn(() => [])
      constructor(cb: IntersectionObserverCallback) {
        setTimeout(() => {
          cb([{ isIntersecting: signalIntersecting } as unknown as IntersectionObserverEntry], this as unknown as IntersectionObserver)
        }, 0)
      }
    } as unknown as typeof IntersectionObserver
  }

  it('scrolls to bottom when isAtBottom is true and lines change', () => {
    vi.useFakeTimers()
    setupObserver(true)
    const scrollToSpy = vi.spyOn(Element.prototype, 'scrollTo')
    const { rerender } = render(<CopilotChat lines={[]} isStreaming={false} />)

    act(() => { vi.advanceTimersByTime(10) })

    rerender(<CopilotChat lines={[{ id: 2, content: 'new msg', type: 'output' }]} isStreaming={false} />)
    expect(scrollToSpy).toHaveBeenCalled()
    scrollToSpy.mockRestore()
    vi.useRealTimers()
  })

  it('shows scroll button when not at bottom and clicking it scrolls down', () => {
    vi.useFakeTimers()
    setupObserver(false)
    vi.spyOn(Element.prototype, 'scrollTo')
    render(<CopilotChat lines={[{ id: 1, content: 'msg', type: 'output' }]} isStreaming={false} />)

    act(() => { vi.advanceTimersByTime(10) })

    const btn = document.querySelector('button[aria-label="Scolla verso il basso"]')
    expect(btn).not.toBeNull()
    btn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(Element.prototype.scrollTo).toHaveBeenCalled()
    vi.useRealTimers()
  })
})
