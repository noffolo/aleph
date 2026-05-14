import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

// --- Mocks ---

vi.mock('lucide-react', () => ({
  PieChart: (props: React.SVGProps<SVGSVGElement>) => (
    <svg {...props} data-testid="pie-chart-icon" />
  ),
}))

import { ToolResultDisplay } from '../ToolResultDisplay'

// --- Tests ---

describe('ToolResultDisplay', () => {
  // ── Empty / falsy result ──────────────────────────────────────────────

  it('renders empty state when result is undefined', () => {
    render(<ToolResultDisplay result={undefined} />)
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  it('renders empty state when result is null', () => {
    render(<ToolResultDisplay result={null} />)
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  it('renders empty state when result is empty string', () => {
    render(<ToolResultDisplay result="" />)
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  it('renders empty state when result is zero', () => {
    render(<ToolResultDisplay result={0} />)
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  it('renders empty state when result is false', () => {
    render(<ToolResultDisplay result={false} />)
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  // ── String result — valid JSON ─────────────────────────────────────────

  it('parses and displays valid JSON string as structured result', () => {
    const result = JSON.stringify({ name: 'alpha', score: 42 })
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
    expect(screen.getByText(/"name": "alpha"/)).toBeInTheDocument()
    expect(screen.getByText(/"score": 42/)).toBeInTheDocument()
  })

  it('parses valid JSON string with nested objects', () => {
    const result = JSON.stringify({ meta: { page: 1, total: 100 }, items: [1, 2, 3] })
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  // ── String result — invalid JSON ───────────────────────────────────────

  it('displays raw text when result is a non-JSON string', () => {
    render(<ToolResultDisplay result="hello world" />)
    expect(screen.getByText('hello world')).toBeInTheDocument()
  })

  it('displays raw text when result is malformed JSON string', () => {
    render(<ToolResultDisplay result="{ broken: true" />)
    expect(screen.getByText('{ broken: true')).toBeInTheDocument()
  })

  it('displays raw text for a string with special characters', () => {
    render(<ToolResultDisplay result="line1\nline2\tindented" />)
    expect(screen.getByText((content) => content.includes('line1') && content.includes('line2') && content.includes('indented'))).toBeInTheDocument()
  })

  // ── Object result — error branch ───────────────────────────────────────

  it('renders error display when parsed result has error string', () => {
    const result = { error: 'Division by zero' }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    expect(screen.getByText('Division by zero')).toBeInTheDocument()
  })

  it('renders error display when parsed result has error object', () => {
    const result = { error: { code: 500, message: 'Internal Server Error' } }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    // Error object is JSON.stringified
    expect(screen.getByText(/"code": 500/)).toBeInTheDocument()
  })

  it('renders error display when parsed result has status "error"', () => {
    const result = { status: 'error', message: 'Something went wrong' }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    // No error field, so fallback is undefined → should display nothing after the header
    // But the component still renders "Execution Error" header
  })

  it('renders error display when result has both error and status error', () => {
    const result = { error: 'Timeout', status: 'error' }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    expect(screen.getByText('Timeout')).toBeInTheDocument()
  })

  // ── Object result — chart_data branch ──────────────────────────────────

  it('renders chart placeholder when result has chart_data', () => {
    const result = { chart_data: [{ x: 1, y: 10 }, { x: 2, y: 20 }] }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Visualizzazione Grafica')).toBeInTheDocument()
    expect(screen.getByText('PLACEHOLDER CHART')).toBeInTheDocument()
    expect(screen.getByText('Dati rilevati: 2')).toBeInTheDocument()
  })

  it('renders chart placeholder with "Oggetto" for non-array chart_data', () => {
    const result = { chart_data: { type: 'line', config: {} } }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Visualizzazione Grafica')).toBeInTheDocument()
    expect(screen.getByText('Dati rilevati: Oggetto')).toBeInTheDocument()
  })

  it('renders chart placeholder for empty chart_data array', () => {
    const result = { chart_data: [] }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Dati rilevati: 0')).toBeInTheDocument()
  })

  // ── Object result from JSON string — error in parsed JSON ─────────────

  it('renders error from JSON-parsed string result', () => {
    const result = JSON.stringify({ error: 'Invalid input' })
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    expect(screen.getByText('Invalid input')).toBeInTheDocument()
  })

  it('renders chart from JSON-parsed string result', () => {
    const result = JSON.stringify({ chart_data: [10, 20, 30] })
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Visualizzazione Grafica')).toBeInTheDocument()
    expect(screen.getByText('Dati rilevati: 3')).toBeInTheDocument()
  })

  // ── Object result — standard structured display ────────────────────────

  it('renders structured result for a plain object', () => {
    const result = { key: 'value', count: 10 }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  it('renders structured result for an array', () => {
    const result = ['alpha', 'beta', 'gamma']
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  it('renders structured result with nested arrays', () => {
    const result = { rows: [[1, 2], [3, 4]] }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  // ── Non-object JSON types ──────────────────────────────────────────────

  it('renders raw number as string', () => {
    render(<ToolResultDisplay result={42} />)
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('renders raw boolean true as string', () => {
    render(<ToolResultDisplay result={true} />)
    expect(screen.getByText('true')).toBeInTheDocument()
  })

  it('renders raw boolean false as string', () => {
    render(<ToolResultDisplay result={false} />)
    // false is falsy, so empty state is shown
    expect(screen.getByText('Nessun risultato prodotto')).toBeInTheDocument()
  })

  // ── JSON string that parses to a non-object ────────────────────────────

  it('parses JSON number string and displays as raw', () => {
    render(<ToolResultDisplay result="42" />)
    // "42" is valid JSON number → isJson=true, but typeof parsed !== 'object' → falls through to raw
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('parses JSON boolean string and displays as raw', () => {
    render(<ToolResultDisplay result="true" />)
    // "true" is valid JSON boolean → isJson=true, but typeof parsed !== 'object' → falls through
    expect(screen.getByText('true')).toBeInTheDocument()
  })

  it('parses JSON null string and displays as raw', () => {
    render(<ToolResultDisplay result="null" />)
    // "null" is valid JSON null → isJson=true, but parsed === null → falls through
    expect(screen.getByText('null')).toBeInTheDocument()
  })

  it('parses JSON array string and displays as structured', () => {
    const result = JSON.stringify([1, 2, 3])
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  // ── Edge cases ─────────────────────────────────────────────────────────

  it('handles object with empty keys gracefully', () => {
    render(<ToolResultDisplay result={{}} />)
    // Empty object → hits structured result path (isJson && typeof parsed === 'object' && parsed !== null)
    // parsed.error is undefined (falsy), parsed.status is undefined
    // 'chart_data' in {} is false
    // Falls through to structured result
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
    expect(screen.getByText('{}')).toBeInTheDocument()
  })

  it('handles result with reserved property names that are not special', () => {
    const result = { chart_data_not_special: [1, 2, 3], status: 'ok' }
    render(<ToolResultDisplay result={result} />)
    // status='ok' is not 'error', chart_data_not_special is not 'chart_data'
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
  })

  it('renders error display when status is exactly "error"', () => {
    const result = { status: 'error' }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
  })

  it('does NOT render error when status is not "error"', () => {
    const result = { status: 'ok', data: [1, 2] }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Structured Result')).toBeInTheDocument()
    expect(screen.queryByText('Execution Error')).not.toBeInTheDocument()
  })

  it('renders large JSON payload with scroll container', () => {
    const items = Array.from({ length: 50 }, (_, i) => ({ id: i, label: `Item ${i}` }))
    render(<ToolResultDisplay result={{ items }} />)
    // Structured result has max-h-[400px] overflow-auto
    const pre = document.querySelector('pre')
    expect(pre).toBeInTheDocument()
    expect(pre?.className).toContain('max-h-')
  })

  it('displays a complex nested error as JSON string', () => {
    const result = { error: { message: 'fail', stack: ['line 1', 'line 2'], code: 'E001' } }
    render(<ToolResultDisplay result={result} />)
    expect(screen.getByText('Execution Error')).toBeInTheDocument()
    expect(screen.getByText(/"message": "fail"/)).toBeInTheDocument()
    expect(screen.getByText(/"code": "E001"/)).toBeInTheDocument()
  })
})
