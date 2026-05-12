import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CopilotView } from '../CopilotView'
import type { ChatMessage } from '../../store/types'
import type { UnifiedCommand } from '../../commands/CommandRegistry'

// --- Mocks ---

vi.mock('../../hooks/useSSE', () => ({
  useSSE: vi.fn(() => ({
    status: 'connected' as const,
    reconnectCount: 0,
  })),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'copilot.showMessageDetail': 'Dettaglio Messaggio',
      'copilot.cancelStream': 'Annulla streaming',
      'copilot.clearChat': 'Pulisci chat',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => ({
  SplitSquareHorizontal: (props: React.SVGProps<SVGSVGElement>) => (
    <svg {...props} data-testid="split-square-icon" />
  ),
}))

vi.mock('../CopilotChat', () => ({
  CopilotChat: ({
    lines,
    isStreaming,
    onMessageClick,
  }: {
    lines: { id: number; type: string; content: string; timestamp: number }[]
    isStreaming: boolean
    onMessageClick: (id: number) => void
  }) => (
    <div data-testid="copilot-chat" data-streaming={isStreaming}>
      {lines.map((line) => (
        <div key={line.id} data-testid="chat-line" onClick={() => onMessageClick(line.id)}>
          {line.content}
        </div>
      ))}
    </div>
  ),
}))

vi.mock('../CopilotSettings', () => ({
  CopilotSettings: ({
    message,
    onClose,
  }: {
    message: ChatMessage | null
    onClose: () => void
  }) => (
    <div data-testid="copilot-settings">
      {message ? <span data-testid="settings-message">{message.content}</span> : null}
      <button data-testid="settings-close" onClick={onClose}>
        Close
      </button>
    </div>
  ),
}))

vi.mock('../ChatSearchBar', () => ({
  ChatSearchBar: ({
    query,
    setQuery,
    matchCount,
  }: {
    query: string
    setQuery: (q: string) => void
    matchCount: number
  }) => (
    <div data-testid="chat-search-bar">
      <input
        data-testid="search-input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <span data-testid="search-match-count">{matchCount}</span>
    </div>
  ),
}))

vi.mock('../terminal', () => ({
  TerminalPrompt: ({
    value,
    onChange,
    onSubmit,
    disabled,
    placeholder,
    'aria-label': ariaLabel,
  }: {
    value: string
    onChange: (val: string) => void
    onSubmit: () => void
    disabled: boolean
    placeholder: string
    'aria-label': string
  }) => (
    <div data-testid="terminal-prompt">
      <input
        data-testid="prompt-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder={placeholder}
        aria-label={ariaLabel}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !disabled && value.trim()) {
            onSubmit()
          }
        }}
      />
    </div>
  ),
  escapeHtml: (s: string) => s,
}))

vi.mock('../../commands', () => ({
  COMMAND_REGISTRY: [
    { name: '/help', description: 'Show commands', action: 'SWITCH_COPILOT' },
    { name: '/clear', description: 'Clear chat', action: 'CLEAR_CHAT' },
    { name: '/settings', description: 'Open settings', action: 'SHOW_INLINE', target: 'settings' },
    { name: '/agents', description: 'Gestisci agenti', action: 'SHOW_INLINE', target: 'agent' },
    { name: '/tools', description: 'Tools panel', action: 'SHOW_INLINE', target: 'tool' },
    { name: '/explore', description: 'Explore data', action: 'SHOW_INLINE', target: 'explore' },
    { name: '/unknown', description: 'Unknown command', action: 'SHOW_INLINE', target: 'unknown' },
  ] as UnifiedCommand[],
}))

// Import mocked useSSE to control return values in tests
import { useSSE } from '../../hooks/useSSE'
const mockUseSSE = useSSE as ReturnType<typeof vi.fn>

// --- Test helpers ---

interface Agent {
  id: string
  name: string
  model: string
}

function makeMsg(overrides?: Partial<ChatMessage>): ChatMessage {
  return {
    role: 'assistant',
    content: 'Hello world',
    createdAt: Date.now(),
    ...overrides,
  }
}

function makeAgent(id: string, overrides?: Partial<Agent>): Agent {
  return { id, name: `Agent ${id}`, model: 'gpt-4', ...overrides }
}

// --- Tests ---

describe('CopilotView', () => {
  let mockSetSelectedAgent: ReturnType<typeof vi.fn>
  let mockSetInput: ReturnType<typeof vi.fn>
  let mockOnSend: ReturnType<typeof vi.fn>
  let mockOnCancelStream: ReturnType<typeof vi.fn>
  let mockOnConfirmAction: ReturnType<typeof vi.fn>
  let mockOnClearChat: ReturnType<typeof vi.fn>

  function defaultProps() {
    return {
      agents: [makeAgent('1'), makeAgent('2', { name: 'GPT Agent', model: 'gpt-4o' })],
      selectedAgent: '1',
      setSelectedAgent: mockSetSelectedAgent as (id: string) => void,
      chat: [] as ChatMessage[],
      input: '',
      setInput: mockSetInput as (val: string) => void,
      onSend: mockOnSend as () => void,
      isStreaming: false,
      onCancelStream: mockOnCancelStream as () => void,
      onConfirmAction: mockOnConfirmAction as (approved: boolean) => void,
      onClearChat: mockOnClearChat as () => void,
    }
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockSetSelectedAgent = vi.fn()
    mockSetInput = vi.fn()
    mockOnSend = vi.fn()
    mockOnCancelStream = vi.fn()
    mockOnConfirmAction = vi.fn()
    mockOnClearChat = vi.fn()
    mockUseSSE.mockReturnValue({ status: 'connected', reconnectCount: 0 })
  })

  it('renders the COPILOT header', () => {
    render(<CopilotView {...defaultProps()} />)
    expect(screen.getByText('COPILOT')).toBeInTheDocument()
  })

  it('has region role with aria-label Chat', () => {
    render(<CopilotView {...defaultProps()} />)
    expect(screen.getByRole('region', { name: 'Chat' })).toBeInTheDocument()
  })

  it('renders agent select dropdown with correct options', () => {
    render(<CopilotView {...defaultProps()} />)
    const select = screen.getByRole('combobox') as HTMLSelectElement
    expect(select).toBeInTheDocument()
    expect(select.value).toBe('1')
    expect(screen.getByText('Agent 1 (gpt-4)')).toBeInTheDocument()
    expect(screen.getByText('GPT Agent (gpt-4o)')).toBeInTheDocument()
  })

  it('calls setSelectedAgent when agent is changed', () => {
    render(<CopilotView {...defaultProps()} />)
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: '2' } })
    expect(mockSetSelectedAgent).toHaveBeenCalledWith('2')
  })

  it('disables agent select when isStreaming is true', () => {
    render(<CopilotView {...defaultProps()} isStreaming={true} />)
    const select = screen.getByRole('combobox') as HTMLSelectElement
    expect(select.disabled).toBe(true)
  })

  it('renders messages via CopilotChat', () => {
    const chat = [makeMsg(), makeMsg({ role: 'user', content: 'Hi' })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    expect(screen.getByTestId('copilot-chat')).toBeInTheDocument()
    const lines = screen.getAllByTestId('chat-line')
    expect(lines).toHaveLength(2)
    expect(lines[1]).toHaveTextContent('Hi')
  })

  it('renders TerminalPrompt with correct props', () => {
    render(<CopilotView {...defaultProps()} input="test input" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput).toBeInTheDocument()
    expect(promptInput.value).toBe('test input')
    expect(promptInput.getAttribute('aria-label')).toBe('Message input')
  })

  it('calls onSend when Enter is pressed with non-empty value', () => {
    render(<CopilotView {...defaultProps()} input="hello" />)
    const promptInput = screen.getByTestId('prompt-input')
    fireEvent.keyDown(promptInput, { key: 'Enter' })
    expect(mockOnSend).toHaveBeenCalledTimes(1)
  })

  it('does not call onSend when input is empty', () => {
    render(<CopilotView {...defaultProps()} input="" />)
    const promptInput = screen.getByTestId('prompt-input')
    fireEvent.keyDown(promptInput, { key: 'Enter' })
    expect(mockOnSend).not.toHaveBeenCalled()
  })

  it('disables TerminalPrompt when isStreaming', () => {
    render(<CopilotView {...defaultProps()} isStreaming={true} />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.disabled).toBe(true)
  })

  it('disables TerminalPrompt when no agent is selected', () => {
    render(<CopilotView {...defaultProps()} selectedAgent="" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.disabled).toBe(true)
  })

  it('shows placeholder to select agent when no agent selected', () => {
    render(<CopilotView {...defaultProps()} selectedAgent="" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.placeholder).toBe('seleziona un agente...')
  })

  it('shows message placeholder when agent is selected', () => {
    render(<CopilotView {...defaultProps()} selectedAgent="1" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.placeholder).toBe('scrivi un messaggio o /comando...')
  })

  it('shows SSE status dot as connected (green)', () => {
    mockUseSSE.mockReturnValue({ status: 'connected', reconnectCount: 0 })
    render(<CopilotView {...defaultProps()} />)
    const dots = document.querySelectorAll('.bg-success')
    expect(dots.length).toBeGreaterThanOrEqual(1)
  })

  it('shows SSE status dot as reconnecting (yellow)', () => {
    mockUseSSE.mockReturnValue({ status: 'reconnecting', reconnectCount: 0 })
    render(<CopilotView {...defaultProps()} />)
    const dots = document.querySelectorAll('.bg-yellow-400')
    expect(dots.length).toBeGreaterThanOrEqual(1)
  })

  it('shows SSE status dot as disconnected (red)', () => {
    mockUseSSE.mockReturnValue({ status: 'disconnected', reconnectCount: 0 })
    render(<CopilotView {...defaultProps()} />)
    const dots = document.querySelectorAll('.bg-danger')
    expect(dots.length).toBeGreaterThanOrEqual(1)
  })

  it('shows reconnectCount badge when reconnectCount > 0', () => {
    mockUseSSE.mockReturnValue({ status: 'connected', reconnectCount: 3 })
    render(<CopilotView {...defaultProps()} />)
    expect(screen.getByText('\u00d73')).toBeInTheDocument()
  })

  it('does not show reconnectCount badge when reconnectCount is 0', () => {
    mockUseSSE.mockReturnValue({ status: 'connected', reconnectCount: 0 })
    render(<CopilotView {...defaultProps()} />)
    expect(screen.queryByText(/^\u00d7/)).not.toBeInTheDocument()
  })

  it('renders SSE status text in footer', () => {
    mockUseSSE.mockReturnValue({ status: 'reconnecting', reconnectCount: 0 })
    render(<CopilotView {...defaultProps()} />)
    expect(screen.getByText('SSE: reconnecting')).toBeInTheDocument()
  })

  it('shows STOP button when isStreaming is true', () => {
    render(<CopilotView {...defaultProps()} isStreaming={true} chat={[makeMsg()]} />)
    expect(screen.getByText('\u23f9 STOP')).toBeInTheDocument()
  })

  it('calls onCancelStream when STOP is clicked', () => {
    render(<CopilotView {...defaultProps()} isStreaming={true} chat={[makeMsg()]} />)
    fireEvent.click(screen.getByText('\u23f9 STOP'))
    expect(mockOnCancelStream).toHaveBeenCalledTimes(1)
  })

  it('does not show STOP button when not streaming', () => {
    render(<CopilotView {...defaultProps()} chat={[makeMsg()]} />)
    expect(screen.queryByText('\u23f9 STOP')).not.toBeInTheDocument()
  })

  it('shows PULISCI button when chat has messages', () => {
    render(<CopilotView {...defaultProps()} chat={[makeMsg()]} />)
    expect(screen.getByText('PULISCI')).toBeInTheDocument()
  })

  it('calls onClearChat when PULISCI is clicked', () => {
    render(<CopilotView {...defaultProps()} chat={[makeMsg()]} />)
    fireEvent.click(screen.getByText('PULISCI'))
    expect(mockOnClearChat).toHaveBeenCalledTimes(1)
  })

  it('does not show PULISCI button when chat is empty', () => {
    render(<CopilotView {...defaultProps()} chat={[]} />)
    expect(screen.queryByText('PULISCI')).not.toBeInTheDocument()
  })

  it('toggles CopilotSettings panel on SplitSquareHorizontal click', () => {
    render(<CopilotView {...defaultProps()} />)
    expect(screen.queryByTestId('copilot-settings')).not.toBeInTheDocument()
    const splitBtn = screen.getByLabelText('Dettaglio Messaggio')
    fireEvent.click(splitBtn)
    expect(screen.getByTestId('copilot-settings')).toBeInTheDocument()
    fireEvent.click(splitBtn)
    expect(screen.queryByTestId('copilot-settings')).not.toBeInTheDocument()
  })

  it('passes selected message to CopilotSettings', () => {
    const chat = [makeMsg({ role: 'user', content: 'first' }), makeMsg({ content: 'second' })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    const splitBtn = screen.getByLabelText('Dettaglio Messaggio')
    fireEvent.click(splitBtn)
    expect(screen.queryByTestId('settings-message')).not.toBeInTheDocument()
    const lines = screen.getAllByTestId('chat-line')
    fireEvent.click(lines[1])
    expect(screen.getByTestId('settings-message')).toHaveTextContent('second')
  })

  it('shows APPROVA and RIFIUTA buttons when chat has requiresConfirmation', () => {
    const chat = [makeMsg({ requiresConfirmation: true, content: 'Approve this?' })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    expect(screen.getByText('APPROVA')).toBeInTheDocument()
    expect(screen.getByText('RIFIUTA')).toBeInTheDocument()
  })

  it('calls onConfirmAction(true) when APPROVA is clicked', () => {
    const chat = [makeMsg({ requiresConfirmation: true })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    fireEvent.click(screen.getByText('APPROVA'))
    expect(mockOnConfirmAction).toHaveBeenCalledWith(true)
  })

  it('calls onConfirmAction(false) when RIFIUTA is clicked', () => {
    const chat = [makeMsg({ requiresConfirmation: true })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    fireEvent.click(screen.getByText('RIFIUTA'))
    expect(mockOnConfirmAction).toHaveBeenCalledWith(false)
  })

  it('hides confirmation bar when isStreaming is true', () => {
    const chat = [makeMsg({ requiresConfirmation: true })]
    render(<CopilotView {...defaultProps()} chat={chat} isStreaming={true} />)
    expect(screen.queryByText('APPROVA')).not.toBeInTheDocument()
  })

  it('does not show confirmation bar when no message requires confirmation', () => {
    render(<CopilotView {...defaultProps()} chat={[makeMsg()]} />)
    expect(screen.queryByText('APPROVA')).not.toBeInTheDocument()
  })

  it('shows command dropdown when input starts with /', () => {
    render(<CopilotView {...defaultProps()} input="/hel" />)
    expect(screen.getByText('Comandi Disponibili')).toBeInTheDocument()
    expect(screen.getByText('/help')).toBeInTheDocument()
  })

  it('filters commands by input after slash', () => {
    render(<CopilotView {...defaultProps()} input="/ag" />)
    expect(screen.getByText('Comandi Disponibili')).toBeInTheDocument()
    expect(screen.getByText('/agents')).toBeInTheDocument()
    expect(screen.queryByText('/help')).not.toBeInTheDocument()
  })

  it('shows "Nessun comando trovato" when no command matches', () => {
    render(<CopilotView {...defaultProps()} input="/zzz" />)
    expect(screen.getByText('Nessun comando trovato')).toBeInTheDocument()
  })

  it('does not show command dropdown when input does not start with /', () => {
    render(<CopilotView {...defaultProps()} input="hello" />)
    expect(screen.queryByText('Comandi Disponibili')).not.toBeInTheDocument()
  })

  it('selects a command and updates input on command click', () => {
    render(<CopilotView {...defaultProps()} input="/hel" />)
    fireEvent.click(screen.getByText('/help'))
    expect(mockSetInput).toHaveBeenCalledWith('/help ')
  })

  it('renders ChatSearchBar with matchCount', () => {
    const chat = [makeMsg({ content: 'alpha' }), makeMsg({ content: 'beta' })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    expect(screen.getByTestId('chat-search-bar')).toBeInTheDocument()
  })

  it('updates matchCount based on search query', () => {
    const chat = [makeMsg({ content: 'hello world' }), makeMsg({ content: 'goodbye' })]
    render(<CopilotView {...defaultProps()} chat={chat} />)
    const searchInput = screen.getByTestId('search-input')
    fireEvent.change(searchInput, { target: { value: 'hello' } })
    expect(screen.getByTestId('search-match-count')).toHaveTextContent('1')
  })

  it('disables prompt when isStreaming is true', () => {
    render(<CopilotView {...defaultProps()} isStreaming={true} selectedAgent="1" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.disabled).toBe(true)
  })

  it('disables prompt when no agent selected', () => {
    render(<CopilotView {...defaultProps()} selectedAgent="" />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.disabled).toBe(true)
  })

  it('enables prompt when agent selected and not streaming', () => {
    render(<CopilotView {...defaultProps()} selectedAgent="1" isStreaming={false} />)
    const promptInput = screen.getByTestId('prompt-input') as HTMLInputElement
    expect(promptInput.disabled).toBe(false)
  })
})
