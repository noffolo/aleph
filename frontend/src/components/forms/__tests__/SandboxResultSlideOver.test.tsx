import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SandboxResultSlideOver } from '../SandboxResultSlideOver'

describe('SandboxResultSlideOver', () => {
  it('renders exit code', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 0, stdout: 'success', stderr: '', metricsJson: '{}' }} />)
    expect(screen.getByText('Exit Code: 0')).toBeInTheDocument()
  })

  it('renders N/A when no result', () => {
    render(<SandboxResultSlideOver />)
    expect(screen.getByText('Exit Code: N/A')).toBeInTheDocument()
  })

  it('renders stdout when present', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 0, stdout: 'Hello World', stderr: '' }} />)
    expect(screen.getByText('Hello World')).toBeInTheDocument()
  })

  it('renders stderr when present', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 1, stdout: '', stderr: 'Error occurred' }} />)
    expect(screen.getByText('Error occurred')).toBeInTheDocument()
  })

  it('applies success class for exit code 0', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 0, stdout: '', stderr: '' }} />)
    const exitCodeEl = screen.getByText('Exit Code: 0')
    expect(exitCodeEl).toHaveClass('text-success')
  })

  it('applies danger class for non-zero exit code', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 1, stdout: '', stderr: '' }} />)
    const exitCodeEl = screen.getByText('Exit Code: 1')
    expect(exitCodeEl).toHaveClass('text-danger')
  })

  it('renders metrics when metricsJson provided', () => {
    render(<SandboxResultSlideOver result={{ exitCode: 0, stdout: '', stderr: '', metricsJson: '{"cpu": 0.5}' }} />)
    expect(screen.getByText('Metrics: {"cpu": 0.5}')).toBeInTheDocument()
  })
})
