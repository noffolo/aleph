import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AssetDetailSlideOver } from '../AssetDetailSlideOver'

const mockAssets = [
  { id: 'a1', name: 'Report.pdf', type: 'pdf', createdAt: 1700000000 },
  { id: 'a2', name: 'Data.csv', type: 'csv', createdAt: 1700001000 },
]

vi.mock('../../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state = { assets: mockAssets }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useLibraryActions', () => ({
  useLibraryActions: () => ({
    onGetAssetContent: vi.fn(),
    onGeneratePdf: vi.fn(),
  }),
}))

describe('AssetDetailSlideOver', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders asset name as title', () => {
    render(<AssetDetailSlideOver assetId="a1" onClose={mockOnClose} />)
    expect(screen.getByText('Report.pdf')).toBeInTheDocument()
  })

  it('renders asset type', () => {
    render(<AssetDetailSlideOver assetId="a1" onClose={mockOnClose} />)
    expect(screen.getByText('Asset Type: pdf')).toBeInTheDocument()
  })

  it('returns null for non-existent asset', () => {
    const { container } = render(<AssetDetailSlideOver assetId="nonexistent" onClose={mockOnClose} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders Vedi Contenuto and Genera PDF buttons', () => {
    render(<AssetDetailSlideOver assetId="a1" onClose={mockOnClose} />)
    expect(screen.getByText('Vedi Contenuto')).toBeInTheDocument()
    expect(screen.getByText('Genera PDF')).toBeInTheDocument()
  })

  it('renders Chiudi button', () => {
    render(<AssetDetailSlideOver assetId="a1" onClose={mockOnClose} />)
    expect(screen.getByText('Chiudi')).toBeInTheDocument()
  })

  it('calls onClose on Chiudi click', () => {
    render(<AssetDetailSlideOver assetId="a1" onClose={mockOnClose} />)
    fireEvent.click(screen.getByText('Chiudi'))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })
})
