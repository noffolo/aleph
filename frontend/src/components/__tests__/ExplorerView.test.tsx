import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ExplorerView } from '../ExplorerView'

vi.mock('../../i18n', () => ({
  t: (key: string) => key,
}))

vi.mock('../../lib/AlephTable', () => ({
  AlephTable: ({ columns, rows }: { columns: string[]; rows: unknown[] }) => (
    <div data-testid="aleph-table">{rows.length} rows</div>
  ),
}))

vi.mock('../../lib/AlephMap', () => ({
  AlephMap: () => <div data-testid="aleph-map">map</div>,
}))

vi.mock('../../lib/AlephTimeline', () => ({
  AlephTimeline: () => <div data-testid="aleph-timeline">timeline</div>,
}))

vi.mock('../../lib/AlephGraph', () => ({
  AlephGraph: () => <div data-testid="aleph-graph">graph</div>,
}))

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">loading</div>,
}))

vi.mock('lucide-react', () => ({
  Search: () => null,
  Table: () => null,
  Map: () => null,
  Clock: () => null,
  Share2: () => null,
}))

describe('ExplorerView', () => {
  const mockSetSelectedObject = vi.fn()
  const mockSetSearchQuery = vi.fn()
  const mockSetActiveView = vi.fn()
  const mockOnRowClick = vi.fn()

  const defaultProps = {
    availableObjects: ['users', 'orders', 'events'],
    selectedObject: 'users',
    setSelectedObject: mockSetSelectedObject,
    searchQuery: '',
    setSearchQuery: mockSetSearchQuery,
    activeView: 'table',
    setActiveView: mockSetActiveView,
    data: { columns: ['id', 'name'], rows: [{ id: 1, name: 'test' }], sql: 'SELECT * FROM users' },
    onRowClick: mockOnRowClick,
    isLoading: false,
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders available object tabs', () => {
    render(<ExplorerView {...defaultProps} />)
    expect(screen.getByText('users')).toBeInTheDocument()
    expect(screen.getByText('orders')).toBeInTheDocument()
    expect(screen.getByText('events')).toBeInTheDocument()
  })

  it('calls setSelectedObject on tab click', () => {
    render(<ExplorerView {...defaultProps} />)
    fireEvent.click(screen.getByText('orders'))
    expect(mockSetSelectedObject).toHaveBeenCalledWith('orders')
  })

  it('renders search input', () => {
    render(<ExplorerView {...defaultProps} />)
    const input = screen.getByPlaceholderText('Cerca in users...')
    expect(input).toBeInTheDocument()
  })

  it('calls setSearchQuery on search input change', () => {
    render(<ExplorerView {...defaultProps} />)
    const input = screen.getByPlaceholderText('Cerca in users...')
    fireEvent.change(input, { target: { value: 'test query' } })
    expect(mockSetSearchQuery).toHaveBeenCalledWith('test query')
  })

  it('renders table view by default', () => {
    render(<ExplorerView {...defaultProps} />)
    expect(screen.getByTestId('aleph-table')).toBeInTheDocument()
  })

  it('switches to map view', () => {
    render(<ExplorerView {...defaultProps} activeView="map" />)
    expect(screen.getByTestId('aleph-map')).toBeInTheDocument()
  })

  it('switches to timeline view', () => {
    render(<ExplorerView {...defaultProps} activeView="timeline" />)
    expect(screen.getByTestId('aleph-timeline')).toBeInTheDocument()
  })

  it('switches to graph view', () => {
    render(<ExplorerView {...defaultProps} activeView="graph" />)
    expect(screen.getByTestId('aleph-graph')).toBeInTheDocument()
  })

  it('calls setActiveView on view button click', () => {
    render(<ExplorerView {...defaultProps} />)
    const mapButton = screen.getByLabelText('explorer.view.map')
    fireEvent.click(mapButton)
    expect(mockSetActiveView).toHaveBeenCalledWith('map')
  })

  it('renders skeleton when isLoading', () => {
    render(<ExplorerView {...defaultProps} isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders SQL preview when data has sql', () => {
    render(<ExplorerView {...defaultProps} />)
    expect(screen.getByText('SELECT * FROM users')).toBeInTheDocument()
  })

  it('does not render SQL preview when data has no sql', () => {
    render(<ExplorerView {...defaultProps} data={{ columns: [], rows: [] }} />)
    expect(screen.queryByText(/SELECT/)).not.toBeInTheDocument()
  })

  it('renders empty state with no objects', () => {
    render(<ExplorerView {...defaultProps} availableObjects={[]} selectedObject="" />)
    expect(screen.getByPlaceholderText(/Cerca in/)).toBeInTheDocument()
  })

  it('calls setActiveView on timeline button click', () => {
    render(<ExplorerView {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('explorer.view.timeline'))
    expect(mockSetActiveView).toHaveBeenCalledWith('timeline')
  })

  it('calls setActiveView on graph button click', () => {
    render(<ExplorerView {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('explorer.view.graph'))
    expect(mockSetActiveView).toHaveBeenCalledWith('graph')
  })
})
