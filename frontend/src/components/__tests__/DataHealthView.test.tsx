import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DataHealthView } from '../DataHealthView'

interface ColumnStats {
  columnName: string
  min: string
  max: string
  count: bigint | number
  uniqueCount: bigint | number
  topValues: Record<string, bigint | number>
}

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">loading</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

vi.mock('lucide-react', () => ({
  BarChart3: () => null,
  Hash: () => null,
  Activity: () => null,
}))

describe('DataHealthView', () => {
  it('renders column name headers', () => {
    const stats: ColumnStats[] = [
      {
        columnName: 'age',
        min: '18',
        max: '65',
        count: 1000n,
        uniqueCount: 50n,
        topValues: { '25': 200n, '30': 180n, '22': 150n, '35': 120n, '28': 100n },
      },
      {
        columnName: 'city',
        min: 'Milano',
        max: 'Roma',
        count: 500n,
        uniqueCount: 10n,
        topValues: { 'Roma': 250n, 'Milano': 200n },
      },
    ]

    render(<DataHealthView stats={stats} />)
    expect(screen.getByText('age')).toBeInTheDocument()
    expect(screen.getByText('city')).toBeInTheDocument()
  })

  it('renders count and unique count labels', () => {
    const stats: ColumnStats[] = [
      {
        columnName: 'age',
        min: '18',
        max: '65',
        count: 1000n,
        uniqueCount: 50n,
        topValues: { '25': 200n, '30': 180n, '22': 150n, '35': 120n, '28': 100n },
      },
      {
        columnName: 'city',
        min: 'Milano',
        max: 'Roma',
        count: 500n,
        uniqueCount: 10n,
        topValues: { 'Roma': 250n, 'Milano': 200n },
      },
    ]
    render(<DataHealthView stats={stats} />)
    const uniciLabels = screen.getAllByText('Unici')
    expect(uniciLabels.length).toBe(2)
    const recordLabels = screen.getAllByText('Record')
    expect(recordLabels.length).toBe(2)
  })

  it('renders distribution labels', () => {
    const stats: ColumnStats[] = [
      {
        columnName: 'age',
        min: '18',
        max: '65',
        count: 1000n,
        uniqueCount: 50n,
        topValues: { '25': 200n, '30': 180n, '22': 150n, '35': 120n, '28': 100n },
      },
    ]
    render(<DataHealthView stats={stats} />)
    const distLabels = screen.getAllByText('Distribuzione Top 5')
    expect(distLabels.length).toBe(1)
  })

  it('renders MIN/MAX values', () => {
    const stats: ColumnStats[] = [
      {
        columnName: 'age',
        min: '18',
        max: '65',
        count: 1000n,
        uniqueCount: 50n,
        topValues: { '25': 200n, '30': 180n, '22': 150n, '35': 120n, '28': 100n },
      },
    ]
    render(<DataHealthView stats={stats} />)
    expect(screen.getByText('MIN: 18')).toBeInTheDocument()
    expect(screen.getByText('MAX: 65')).toBeInTheDocument()
  })

  it('renders empty state when no stats', () => {
    render(<DataHealthView stats={[]} />)
    expect(screen.getByText('Seleziona un oggetto per analizzare la salute dei dati')).toBeInTheDocument()
  })

  it('renders skeleton when loading', () => {
    render(<DataHealthView stats={[]} isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when provided', () => {
    render(<DataHealthView stats={[]} error="Fetch error" />)
    expect(screen.getByTestId('inline-error')).toBeInTheDocument()
    expect(screen.getByText('Fetch error')).toBeInTheDocument()
  })

  it('handles BigInt count values', () => {
    const stats: ColumnStats[] = [
      {
        columnName: 'city',
        min: 'Milano',
        max: 'Roma',
        count: 500n,
        uniqueCount: 10n,
        topValues: { 'Roma': 250n, 'Milano': 200n },
      },
    ]
    render(<DataHealthView stats={stats} />)
    const milanoElements = screen.getAllByText(/Milano/)
    expect(milanoElements.length).toBeGreaterThanOrEqual(1)
  })
})
