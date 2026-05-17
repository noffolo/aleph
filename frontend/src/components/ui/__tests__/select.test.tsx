import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectItem,
  SelectGroup,
  SelectLabel,
  SelectSeparator,
} from '../select'

function SelectFixture({ children }: { children: React.ReactNode }) {
  return <Select>{children}</Select>
}

describe('Select', () => {
  it('renders a select component', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger>
          <SelectValue placeholder="Choose..." />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toBeInTheDocument()
  })
})

describe('SelectTrigger', () => {
  it('renders with data-slot="select-trigger"', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger>
          <SelectValue placeholder="Select..." />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toBeInTheDocument()
  })

  it('renders with default size', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger>
          <SelectValue placeholder="Select..." />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toHaveAttribute('data-size', 'default')
  })

  it('renders with sm size', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger size="sm">
          <SelectValue placeholder="Small" />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toHaveAttribute('data-size', 'sm')
  })

  it('applies custom className', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger className="my-trigger">
          <SelectValue placeholder="Select..." />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toHaveClass('my-trigger')
  })

  it('renders as disabled', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger disabled>
          <SelectValue placeholder="Disabled" />
        </SelectTrigger>
      </SelectFixture>
    )
    const trigger = container.querySelector('[data-slot="select-trigger"]')
    expect(trigger).toHaveClass('disabled:cursor-not-allowed')
  })
})

describe('SelectValue', () => {
  it('renders with placeholder text', () => {
    const { container } = render(
      <SelectFixture>
        <SelectTrigger>
          <SelectValue placeholder="Pick one" />
        </SelectTrigger>
      </SelectFixture>
    )
    const value = container.querySelector('[data-slot="select-value"]')
    expect(value).toBeInTheDocument()
  })
})

describe('SelectItem', () => {
  it('renders a select item inside Select root', () => {
    const { container } = render(
      <SelectFixture>
        <SelectItem value="option1">Option 1</SelectItem>
      </SelectFixture>
    )
    const item = container.querySelector('[data-slot="select-item"]')
    expect(item).toBeInTheDocument()
    expect(screen.getByText('Option 1')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <SelectFixture>
        <SelectItem className="my-item" value="x">X</SelectItem>
      </SelectFixture>
    )
    const item = container.querySelector('[data-slot="select-item"]')
    expect(item).toHaveClass('my-item')
  })
})

describe('SelectGroup', () => {
  it('renders a group with data-slot="select-group"', () => {
    const { container } = render(
      <SelectFixture>
        <SelectGroup />
      </SelectFixture>
    )
    const group = container.querySelector('[data-slot="select-group"]')
    expect(group).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <SelectFixture>
        <SelectGroup className="my-group" />
      </SelectFixture>
    )
    const group = container.querySelector('[data-slot="select-group"]')
    expect(group).toHaveClass('my-group')
  })
})

describe('SelectLabel', () => {
  it('renders a label inside SelectGroup and Select root', () => {
    render(
      <SelectFixture>
        <SelectGroup>
          <SelectLabel>My Label</SelectLabel>
        </SelectGroup>
      </SelectFixture>
    )
    expect(screen.getByText('My Label')).toBeInTheDocument()
  })

  it('applies custom className to label', () => {
    render(
      <SelectFixture>
        <SelectGroup>
          <SelectLabel className="my-label">Styled</SelectLabel>
        </SelectGroup>
      </SelectFixture>
    )
    const label = screen.getByText('Styled')
    expect(label.className).toContain('my-label')
  })
})

describe('SelectSeparator', () => {
  it('renders a separator inside Select root', () => {
    const { container } = render(
      <SelectFixture>
        <SelectSeparator />
      </SelectFixture>
    )
    const sep = container.querySelector('[data-slot="select-separator"]')
    expect(sep).toBeInTheDocument()
  })

  it('applies custom className to separator', () => {
    const { container } = render(
      <SelectFixture>
        <SelectSeparator className="my-sep" />
      </SelectFixture>
    )
    const sep = container.querySelector('[data-slot="select-separator"]')
    expect(sep).toHaveClass('my-sep')
  })
})
