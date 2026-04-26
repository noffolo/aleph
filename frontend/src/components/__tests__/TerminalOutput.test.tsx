import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
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
})
