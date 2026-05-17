import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

const mockOnSendAction = vi.fn()
const mockOnConfirmAction = vi.fn()
const mockOnCancelStream = vi.fn()
const mockClearMessages = vi.fn()

const mockStore = () => ({
  agents: [{ id: 'a1', name: 'Test Agent', model: 'gpt-4', provider: 'openai', apiKey: '', baseUrl: '', systemPrompt: '' }],
  selectedAgent: 'a1',
  setSelectedAgent: vi.fn(),
  messages: [],
  isStreaming: false,
  clearMessages: mockClearMessages,
})

vi.mock('../../store/useStore', () => ({
  useStore: vi.fn((sel: (s: ReturnType<typeof mockStore>) => unknown) => sel(mockStore())),
}))

vi.mock('../../hooks/useAppActions', () => ({
  useAppActions: () => ({
    onSend: mockOnSendAction,
    onConfirmAction: mockOnConfirmAction,
    onCancelStream: mockOnCancelStream,
  }),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => key,
}))

vi.mock('../../hooks/useSSE', () => ({
  useSSE: vi.fn(() => ({ status: 'connected' as const, reconnectCount: 0 })),
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = (props: React.SVGProps<SVGSVGElement>) => <svg {...props} data-testid={`icon-${name}`} />
    Comp.displayName = name
    return Comp
  }
  return { SplitSquareHorizontal: Icon('SplitSquareHorizontal') }
})

vi.mock('../CopilotChat', () => ({
  CopilotChat: ({
    lines, isStreaming, onMessageClick,
  }: { lines: { id: number; type: string; content: string; timestamp: number }[]; isStreaming: boolean; onMessageClick: (id: number) => void }) => (
    <div data-testid="copilot-chat" data-streaming={isStreaming}>
      {lines.map((line) => (
        <div key={line.id} data-testid="chat-line" onClick={() => onMessageClick(line.id)}>{line.content}</div>
      ))}
    </div>
  ),
}))

vi.mock('../terminal', () => ({
  TerminalPrompt: ({
    value, onChange, onSubmit, disabled,
  }: { value: string; onChange: (v: string) => void; onSubmit: () => void; disabled: boolean }) => (
    <div data-testid="terminal-prompt">
      <input data-testid="prompt-input" value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled}
        onKeyDown={(e) => { if (e.key === 'Enter' && value.trim()) onSubmit() }} />
    </div>
  ),
  escapeHtml: (s: string) => s,
}))

vi.mock('../CopilotSettings', () => ({
  CopilotSettings: ({ message }: { message: unknown }) => <div data-testid="copilot-settings">{message ? 'active' : 'none'}</div>,
}))

vi.mock('../ChatSearchBar', () => ({
  ChatSearchBar: () => <div data-testid="chat-search-bar" />,
}))

vi.mock('../../commands', () => ({
  COMMAND_REGISTRY: [],
}))

import { TerminalView } from '../terminal/TerminalView'

describe('TerminalView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the aleph-v2 header', () => {
    render(<TerminalView />)
    expect(screen.getByText('aleph-v2')).toBeInTheDocument()
  })

  it('renders "terminal" label', () => {
    render(<TerminalView />)
    expect(screen.getByText('terminal')).toBeInTheDocument()
  })

  it('renders the CopilotView (via CopilotChat)', () => {
    render(<TerminalView />)
    expect(screen.getByTestId('copilot-chat')).toBeInTheDocument()
  })

  it('renders TerminalPrompt', () => {
    render(<TerminalView />)
    expect(screen.getByTestId('terminal-prompt')).toBeInTheDocument()
  })

  it('shows selected agent label', () => {
    render(<TerminalView />)
    expect(screen.getByText('a1')).toBeInTheDocument()
  })

  it('calls onSend with input and clears on Enter', () => {
    render(<TerminalView />)
    const input = screen.getByTestId('prompt-input') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'test message' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(mockOnSendAction).toHaveBeenCalledWith('test message')
  })

  it('does not call onSend with empty input', () => {
    render(<TerminalView />)
    const input = screen.getByTestId('prompt-input') as HTMLInputElement
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(mockOnSendAction).not.toHaveBeenCalled()
  })
})
