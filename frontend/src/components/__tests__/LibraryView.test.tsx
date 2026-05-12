import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { LibraryView } from '../LibraryView'

const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      const state = {
        setSlideOverContent: mockSetSlideOverContent,
        expandedSections: { 'library.list': true },
        toggleSection: vi.fn(),
      }
      if (typeof selector === 'function') return selector(state)
      return state
    }),
    { getState: vi.fn(() => ({ setSlideOverContent: mockSetSlideOverContent })) },
  ),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'library.title': 'Libreria',
      'library.subtitle': 'Asset e report',
      'library.upload': 'Carica',
      'library.dragAndDrop': 'Trascina qui',
      'library.dropToUpload': 'Rilascia per caricare',
      'generic.loadingLower': 'Caricamento...',
    }
    return map[key] ?? key
  },
}))

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">loading</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

vi.mock('lucide-react', () => ({
  Book: () => null,
  FileText: () => null,
  Download: () => null,
  Trash2: () => null,
  Upload: () => null,
  ChevronDown: () => null,
}))

describe('LibraryView', () => {
  const mockOnViewAsset = vi.fn()
  const mockOnDeleteAsset = vi.fn()
  const mockOnGetAssetContent = vi.fn()
  const mockOnGeneratePdf = vi.fn()
  const mockOnUploadAsset = vi.fn()

  const assets = [
    { id: 'a1', name: 'Report Q3', type: 'pdf', createdAt: 1700000000 },
    { id: 'a2', name: 'Data Export', type: 'csv', createdAt: 1700001000 },
  ]

  const buildProps = (overrides: object = {}) => ({
    assets,
    onViewAsset: mockOnViewAsset,
    onDeleteAsset: mockOnDeleteAsset,
    selectedAssetContent: null,
    setSelectedAssetContent: vi.fn(),
    onGetAssetContent: mockOnGetAssetContent,
    onGeneratePdf: mockOnGeneratePdf,
    onUploadAsset: mockOnUploadAsset,
    ...overrides,
  })

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title', () => {
    render(<LibraryView {...buildProps()} />)
    expect(screen.getByText('Libreria')).toBeInTheDocument()
  })

  it('renders asset cards', () => {
    render(<LibraryView {...buildProps()} />)
    expect(screen.getByText('Report Q3')).toBeInTheDocument()
    expect(screen.getByText('Data Export')).toBeInTheDocument()
  })

  it('renders empty state when no assets', () => {
    render(<LibraryView {...buildProps({ assets: [] })} />)
    expect(screen.getByText(/Nessun report generato/)).toBeInTheDocument()
  })

  it('opens confirm SlideOver when delete button clicked', () => {
    render(<LibraryView {...buildProps()} />)
    const deleteBtn = screen.getByLabelText(/Elimina Report Q3/)
    fireEvent.click(deleteBtn)
    // onDeleteAsset should NOT be called yet — only after confirming in the SlideOver
    expect(mockOnDeleteAsset).not.toHaveBeenCalled()
    // setSlideOverContent should have been called with the confirm dialog config
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'confirm',
      title: 'Conferma eliminazione',
      data: {
        message: 'Sei sicuro di voler eliminare questo asset?',
        onConfirm: expect.any(Function),
      },
    })
  })

  it('renders upload button', () => {
    render(<LibraryView {...buildProps()} />)
    expect(screen.getByText('Carica')).toBeInTheDocument()
  })

  it('renders skeleton when loading', () => {
    render(<LibraryView {...buildProps({ assets: [], isLoading: true })} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when provided', () => {
    render(<LibraryView {...buildProps({ assets: [], error: 'Load failed' })} />)
    expect(screen.getByTestId('inline-error')).toBeInTheDocument()
    expect(screen.getByText('Load failed')).toBeInTheDocument()
  })

  it('renders drag and drop area', () => {
    render(<LibraryView {...buildProps({ assets: [] })} />)
    expect(screen.getByText(/Trascina qui/)).toBeInTheDocument()
  })
})
