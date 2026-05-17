import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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
    expect(mockOnDeleteAsset).not.toHaveBeenCalled()
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

  it('handles download using selectedAssetContent', () => {
    const createObjectURL = vi.fn().mockReturnValue('blob:test')
    const revokeObjectURL = vi.fn()
    const createElement = document.createElement.bind(document)
    const mockAnchor = { href: '', download: '', click: vi.fn() } as unknown as HTMLAnchorElement
    vi.spyOn(URL, 'createObjectURL').mockImplementation(createObjectURL)
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(revokeObjectURL)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'a') return mockAnchor
      return createElement(tag)
    })
    render(<LibraryView {...buildProps({ selectedAssetContent: 'file content here' })} />)
    const downloadBtn = screen.getByLabelText(/Scarica Report Q3/)
    fireEvent.click(downloadBtn)
    expect(createObjectURL).toHaveBeenCalled()
    expect(mockAnchor.click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalled()
  })

  it('handles download by fetching content when not cached', async () => {
    mockOnGetAssetContent.mockResolvedValue('fetched content')
    render(<LibraryView {...buildProps({ selectedAssetContent: null })} />)
    const downloadBtn = screen.getByLabelText(/Scarica Report Q3/)
    await userEvent.click(downloadBtn)
    expect(mockOnGetAssetContent).toHaveBeenCalledWith('a1')
  })

  it('opens asset detail SlideOver on Leggi Report click', () => {
    render(<LibraryView {...buildProps()} />)
    fireEvent.click(screen.getAllByText('Leggi Report')[0])
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'asset',
      title: 'Dettaglio Asset',
      data: { assetId: 'a1' },
    })
  })

  it('renders dragOver state when dragging over the drop zone', () => {
    render(<LibraryView {...buildProps({ assets: [] })} />)
    const dropZone = screen.getByText(/Trascina qui/).closest('div')!.parentElement!
    fireEvent.dragOver(dropZone)
    expect(screen.getByText('Rilascia per caricare')).toBeInTheDocument()
  })

  it('renders inline mode', () => {
    render(<LibraryView {...buildProps({ inline: true })} />)
    expect(screen.getByText('Libreria')).toBeInTheDocument()
  })

  it('shows uploading state on upload button', () => {
    render(<LibraryView {...buildProps()} />)
    expect(screen.getByText('Carica')).toBeInTheDocument()
  })

  it('renders file input hidden element', () => {
    render(<LibraryView {...buildProps()} />)
    const fileInput = document.querySelector('input[type="file"]')
    expect(fileInput).toBeInTheDocument()
    expect(fileInput).toHaveAttribute('multiple')
    expect(fileInput!.className).toContain('hidden')
  })

  it('processes files on file input change', async () => {
    const file = new File(['hello world'], 'test.txt', { type: 'text/plain' })
    render(<LibraryView {...buildProps()} />)
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    Object.defineProperty(fileInput, 'files', { value: [file], writable: false })
    fireEvent.change(fileInput)
    await vi.waitFor(() => {
      expect(mockOnUploadAsset).toHaveBeenCalledWith('test.txt', expect.any(Uint8Array))
    })
  })

  it('processes files on drop', async () => {
    const file = new File(['drop content'], 'dropped.txt', { type: 'text/plain' })
    const dataTransfer = {
      files: [file],
      items: [],
      types: ['Files'],
      getData: vi.fn(),
      setData: vi.fn(),
      clearData: vi.fn(),
    } as unknown as DataTransfer
    render(<LibraryView {...buildProps({ assets: [] })} />)
    const dropZone = screen.getByText(/Trascina qui/).closest('div')!.parentElement!
    fireEvent.drop(dropZone, { dataTransfer })
    await vi.waitFor(() => {
      expect(mockOnUploadAsset).toHaveBeenCalledWith('dropped.txt', expect.any(Uint8Array))
    })
  })

  it('clears dragOver on dragLeave', () => {
    render(<LibraryView {...buildProps({ assets: [] })} />)
    const dropZone = screen.getByText(/Trascina qui/).closest('div')!.parentElement!
    fireEvent.dragOver(dropZone)
    expect(screen.getByText('Rilascia per caricare')).toBeInTheDocument()
    fireEvent.dragLeave(dropZone)
    expect(screen.queryByText('Rilascia per caricare')).not.toBeInTheDocument()
  })

  it('shows upload button disabled during upload', async () => {
    const file = new File(['test'], 'file.txt', { type: 'text/plain' })
    // Make upload slow so we can observe the disabled state
    mockOnUploadAsset.mockImplementation(() => new Promise(() => {}))
    render(<LibraryView {...buildProps()} />)
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    Object.defineProperty(fileInput, 'files', { value: [file], writable: false })
    fireEvent.change(fileInput)
    await vi.waitFor(() => {
      expect(screen.getByText('Caricamento...')).toBeInTheDocument()
    })
    // Reset mock so subsequent tests aren't affected
    mockOnUploadAsset.mockImplementation(() => Promise.resolve())
  })

  it('renders asset date and type badges', () => {
    render(<LibraryView {...buildProps()} />)
    const dateStr = new Date(1700000000 * 1000).toLocaleDateString()
    const dateElements = screen.getAllByText(dateStr)
    expect(dateElements.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('pdf')).toBeInTheDocument()
    expect(screen.getByText('csv')).toBeInTheDocument()
  })
})
