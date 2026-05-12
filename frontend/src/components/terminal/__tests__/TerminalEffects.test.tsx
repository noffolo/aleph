import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockState = { enableScanline: false, enableGlow: false, enableFlicker: false }

vi.mock('../../../store/useStore', () => ({
  useStore: vi.fn((selector: (s: typeof mockState) => unknown) => {
    return selector(mockState)
  }),
}))

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(() => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

import { TerminalEffects } from '../TerminalEffects'

describe('TerminalEffects', () => {
  beforeEach(() => {
    mockState.enableScanline = false
    mockState.enableGlow = false
    mockState.enableFlicker = false
  })

  it('renders a fixed overlay with aria-hidden', () => {
    const { container } = render(<TerminalEffects />)
    const overlay = container.firstElementChild
    expect(overlay).toHaveClass('fixed', 'inset-0', 'pointer-events-none')
    expect(overlay).toHaveAttribute('aria-hidden', 'true')
  })

  it('renders no effects when disabled', () => {
    const { container } = render(<TerminalEffects />)
    const html = container.innerHTML
    expect(html).not.toContain('repeating-linear-gradient')
    expect(html).not.toContain('flicker 0.15s')
    expect(html).not.toContain('radial-gradient')
  })

  it('renders scanline when enabled', () => {
    mockState.enableScanline = true
    const { container } = render(<TerminalEffects />)
    expect(container.innerHTML).toContain('repeating-linear-gradient')
  })

  it('renders flicker when enabled', () => {
    mockState.enableFlicker = true
    const { container } = render(<TerminalEffects />)
    expect(container.innerHTML).toContain('flicker 0.15s')
  })

  it('renders glow when enabled', () => {
    mockState.enableGlow = true
    const { container } = render(<TerminalEffects />)
    expect(container.innerHTML).toContain('radial-gradient')
  })

  it('renders all effects when all enabled', () => {
    mockState.enableScanline = true
    mockState.enableGlow = true
    mockState.enableFlicker = true
    const { container } = render(<TerminalEffects />)
    expect(container.innerHTML).toContain('repeating-linear-gradient')
    expect(container.innerHTML).toContain('flicker 0.15s')
    expect(container.innerHTML).toContain('radial-gradient')
  })

  it('applies custom className', () => {
    const { container } = render(<TerminalEffects className="my-effects" />)
    expect(container.firstElementChild).toHaveClass('my-effects')
  })
})
