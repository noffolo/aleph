import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TerminalOutput } from '../terminal/TerminalOutput'
import type { TerminalLine } from '../terminal/TerminalOutput'

describe('TerminalOutput', () => {
  const mockLines: TerminalLine[] = [
    { type: 'input' as const, content: 'ls -la', id: '1' },
    { type: 'output' as const, content: 'total 40K', id: '2' },
    { type: 'error' as const, content: 'Permission denied', id: '3' },
    { type: 'system' as const, content: 'System initialized', id: '4' },
    { type: 'tool' as const, content: 'Fetching data...', id: '5' },
  ]

  it('renders all lines correctly', () => {
    render(<TerminalOutput lines={mockLines} />)

    expect(screen.getByText('ls -la')).toBeInTheDocument()
    expect(screen.getByText('total 40K')).toBeInTheDocument()
    expect(screen.getByText('Permission denied')).toBeInTheDocument()
    expect(screen.getByText('System initialized')).toBeInTheDocument()
    expect(screen.getByText('Fetching data...')).toBeInTheDocument()
  })

  it('renders streaming cursor when isStreaming', () => {
    render(<TerminalOutput lines={[]} isStreaming={true} />)
    expect(screen.getByText('█')).toBeInTheDocument()
  })

  it('hides cursor when isStreaming is false', () => {
    render(<TerminalOutput lines={[]} isStreaming={false} />)
    expect(screen.queryByText('█')).not.toBeInTheDocument()
  })

  it('handles empty lines', () => {
    const { container } = render(<TerminalOutput lines={[]} />)
    expect(container).toBeInTheDocument()
  })

  it('calls onMessageClick when a line is clicked', () => {
    const onMessageClick = vi.fn()
    render(<TerminalOutput lines={mockLines} onMessageClick={onMessageClick} />)
    fireEvent.click(screen.getByText('ls -la'))
    expect(onMessageClick).toHaveBeenCalledWith(0)
  })

  it('renders timestamp when line has timestamp', () => {
    const linesWithTimestamp: TerminalLine[] = [
      { type: 'output' as const, content: 'Done', id: '1', timestamp: 1700000000000 },
    ]
    render(<TerminalOutput lines={linesWithTimestamp} />)
    expect(screen.getByText(/:/)).toBeInTheDocument()
  })

  it('applies correct style class for input type', () => {
    const { container } = render(<TerminalOutput lines={[{ type: 'input' as const, content: 'cmd', id: '1' }]} />)
    expect(container.querySelector('.text-primary')).toBeInTheDocument()
  })

  it('applies correct style class for error type', () => {
    const { container } = render(<TerminalOutput lines={[{ type: 'error' as const, content: 'err', id: '1' }]} />)
    expect(container.querySelector('.text-danger')).toBeInTheDocument()
  })

  it('applies correct style class for tool type', () => {
    const { container } = render(<TerminalOutput lines={[{ type: 'tool' as const, content: 'fetching', id: '1' }]} />)
    expect(container.querySelector('.text-warning')).toBeInTheDocument()
  })

  it('shows lambda prefix for input lines', () => {
    render(<TerminalOutput lines={[{ type: 'input' as const, content: 'cmd', id: '1' }]} />)
    expect(screen.getByText('λ')).toBeInTheDocument()
  })

  it('shows arrow prefix for system lines', () => {
    render(<TerminalOutput lines={[{ type: 'system' as const, content: 'init', id: '1' }]} />)
    expect(screen.getByText('→</')).toBeInTheDocument()
  })

  it('shows gear prefix for tool lines', () => {
    render(<TerminalOutput lines={[{ type: 'tool' as const, content: 'fetching', id: '1' }]} />)
    expect(screen.getByText('⚙')).toBeInTheDocument()
  })

  it('handles streaming effect with output line', () => {
    const rafSpy = vi.spyOn(window, 'requestAnimationFrame').mockImplementation((cb) => {
      setTimeout(cb, 16)
      return 1
    })
    const streamingLines: TerminalLine[] = [
      { type: 'output' as const, content: '<b>streaming</b>', id: '1' },
    ]
    render(<TerminalOutput lines={streamingLines} isStreaming={true} />)
    expect(screen.getByText('█')).toBeInTheDocument()
    rafSpy.mockRestore()
  })
})
