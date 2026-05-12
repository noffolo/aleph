import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../dialog'

describe('Dialog', () => {
  it('renders children inside Dialog root', () => {
    render(
      <Dialog>
        <DialogTrigger>Open</DialogTrigger>
      </Dialog>
    )
    expect(screen.getByText('Open')).toBeInTheDocument()
  })
})

describe('DialogTrigger', () => {
  it('renders trigger inside Dialog root', () => {
    render(
      <Dialog>
        <DialogTrigger>Open Dialog</DialogTrigger>
      </Dialog>
    )
    expect(screen.getByText('Open Dialog')).toBeInTheDocument()
  })
})

describe('DialogHeader', () => {
  it('renders header div with data-slot="dialog-header"', () => {
    const { container } = render(<DialogHeader />)
    const header = container.querySelector('[data-slot="dialog-header"]')
    expect(header).toBeInTheDocument()
    expect(header).toHaveClass('flex', 'flex-col', 'gap-2')
  })

  it('applies custom className', () => {
    const { container } = render(<DialogHeader className="my-header" />)
    const header = container.querySelector('[data-slot="dialog-header"]')
    expect(header).toHaveClass('my-header')
  })
})

describe('DialogFooter', () => {
  it('renders footer div with data-slot="dialog-footer"', () => {
    const { container } = render(<DialogFooter />)
    const footer = container.querySelector('[data-slot="dialog-footer"]')
    expect(footer).toBeInTheDocument()
    expect(footer).toHaveClass('flex', 'flex-col-reverse')
  })

  it('applies custom className', () => {
    const { container } = render(<DialogFooter className="my-footer" />)
    const footer = container.querySelector('[data-slot="dialog-footer"]')
    expect(footer).toHaveClass('my-footer')
  })
})

describe('DialogTitle', () => {
  it('renders title inside Dialog', () => {
    render(
      <Dialog>
        <DialogTitle>My Title</DialogTitle>
      </Dialog>
    )
    expect(screen.getByText('My Title')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <Dialog>
        <DialogTitle className="my-title">Title</DialogTitle>
      </Dialog>
    )
    const title = container.querySelector('[data-slot="dialog-title"]')
    expect(title).toHaveClass('my-title')
  })

  it('has font-heading class', () => {
    const { container } = render(
      <Dialog>
        <DialogTitle>Title</DialogTitle>
      </Dialog>
    )
    const title = container.querySelector('[data-slot="dialog-title"]')
    expect(title).toHaveClass('font-heading', 'text-base', 'font-medium')
  })
})

describe('DialogDescription', () => {
  it('renders description inside Dialog', () => {
    render(
      <Dialog>
        <DialogDescription>Some description</DialogDescription>
      </Dialog>
    )
    expect(screen.getByText('Some description')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <Dialog>
        <DialogDescription className="my-desc">Desc</DialogDescription>
      </Dialog>
    )
    const desc = container.querySelector('[data-slot="dialog-description"]')
    expect(desc).toHaveClass('my-desc')
  })
})

describe('DialogClose', () => {
  it('renders close inside Dialog', () => {
    render(
      <Dialog>
        <DialogClose>Close</DialogClose>
      </Dialog>
    )
    expect(screen.getByText('Close')).toBeInTheDocument()
  })
})

describe('DialogContent', () => {
  it('renders children inside dialog content when open', () => {
    render(
      <Dialog open>
        <DialogContent>
          <span data-testid="inner">Hello Content</span>
        </DialogContent>
      </Dialog>
    )
    expect(document.querySelector('[data-testid="inner"]')).toBeInTheDocument()
  })

  it('renders content with data-slot="dialog-content" when open', () => {
    render(
      <Dialog open>
        <DialogContent>
          <span>Content</span>
        </DialogContent>
      </Dialog>
    )
    expect(document.querySelector('[data-slot="dialog-content"]')).toBeInTheDocument()
  })
})
