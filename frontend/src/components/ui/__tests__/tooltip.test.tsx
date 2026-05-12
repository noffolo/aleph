import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
} from '../tooltip'

function TooltipFixture({ children }: { children: React.ReactNode }) {
  return (
    <TooltipProvider>
      <Tooltip>{children}</Tooltip>
    </TooltipProvider>
  )
}

describe('Tooltip', () => {
  it('renders trigger text within context', () => {
    render(
      <TooltipFixture>
        <TooltipTrigger>Hover me</TooltipTrigger>
      </TooltipFixture>
    )
    expect(screen.getByText('Hover me')).toBeInTheDocument()
  })
})

describe('TooltipProvider', () => {
  it('renders children inside provider', () => {
    render(
      <TooltipProvider>
        <span data-testid="child">test</span>
      </TooltipProvider>
    )
    expect(screen.getByTestId('child')).toBeInTheDocument()
  })
})

describe('TooltipTrigger', () => {
  it('renders trigger within context with correct role', () => {
    const { container } = render(
      <TooltipFixture>
        <TooltipTrigger>Trigger</TooltipTrigger>
      </TooltipFixture>
    )
    const trigger = container.querySelector('[data-slot="tooltip-trigger"]')
    expect(trigger).toBeInTheDocument()
    expect(trigger?.textContent).toBe('Trigger')
  })

  it('renders children inside trigger', () => {
    render(
      <TooltipFixture>
        <TooltipTrigger>
          <span data-testid="trigger-text">Hover</span>
        </TooltipTrigger>
      </TooltipFixture>
    )
    expect(screen.getByTestId('trigger-text')).toBeInTheDocument()
    expect(screen.getByText('Hover')).toBeInTheDocument()
  })
})

describe('TooltipContent', () => {
  it('renders trigger alongside content within context', () => {
    const { container } = render(
      <TooltipFixture>
        <TooltipTrigger>Trigger</TooltipTrigger>
        <TooltipContent>Helpful tooltip</TooltipContent>
      </TooltipFixture>
    )
    const trigger = container.querySelector('[data-slot="tooltip-trigger"]')
    expect(trigger).toBeInTheDocument()
  })
})
