import React, { useRef, useEffect } from 'react'
import { useStore } from '../store/useStore'
import { TerminalPrompt } from './terminal'
import { TerminalOutput, escapeHtml } from './terminal'
import { InlineRenderer } from './terminal/InlineRenderer'
import { InlineErrorBoundary } from './InlineErrorBoundary'

interface ChatMessage {
  role: string
  content: string
  toolCall?: string
  requiresConfirmation?: boolean
  createdAt?: number
}

interface Agent {
  id: string
  name: string
  model: string
}

interface CopilotViewProps {
  agents: Agent[]
  selectedAgent: string
  setSelectedAgent: (id: string) => void
  chat: ChatMessage[]
  input: string
  setInput: (val: string) => void
  onSend: () => void
  isStreaming: boolean
  onCancelStream: () => void
  onConfirmAction: (approved: boolean) => void
  onClearChat: () => void
}

export const CopilotView: React.FC<CopilotViewProps> = ({
  agents, selectedAgent, setSelectedAgent,
  chat, input, setInput, onSend, isStreaming, onCancelStream, onConfirmAction, onClearChat
}) => {
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    scrollRef.current?.scrollTo(0, scrollRef.current.scrollHeight)
  }, [chat, isStreaming])

  const terminalLines = chat.map((msg, i) => ({
    id: i,
    type: msg.role === 'user' ? 'input' as const : msg.toolCall ? 'tool' as const : 'output' as const,
    content: escapeHtml(msg.content || msg.toolCall || ''),
    timestamp: msg.createdAt,
  }))

  return (
    <div className="flex flex-col h-full bg-surface rounded-lg border border-border overflow-hidden">
      <div className="h-9 flex items-center justify-between px-4 border-b border-border shrink-0">
        <div className="flex items-center gap-2">
          <span className="text-primary text-xs font-bold">COPILOT</span>
          <span className="text-textDim text-xs">│</span>
          <select
            value={selectedAgent}
            onChange={(e) => setSelectedAgent(e.target.value)}
            disabled={isStreaming}
            className="bg-transparent text-text text-xs outline-none cursor-pointer disabled:opacity-50"
          >
            <option value="" className="bg-surface text-text">seleziona agente...</option>
            {agents.map(a => (
              <option key={a.id} value={a.id} className="bg-surface text-text">{a.name} ({a.model})</option>
            ))}
          </select>
        </div>
        {chat.length > 0 && (
          <div className="flex items-center gap-4">
            {isStreaming && (
              <button onClick={onCancelStream} className="text-danger hover:text-danger-bright text-xs font-bold transition-colors" title="Interrompi streaming">
                ⏹ STOP
              </button>
            )}
            <button onClick={onClearChat} className="text-textMuted hover:text-danger text-xs transition-colors" title="Pulisci">
              PULISCI
            </button>
          </div>
        )}
      </div>

      <div ref={scrollRef} className="flex-1 overflow-auto">
        <TerminalOutput lines={terminalLines} isStreaming={isStreaming} />
      </div>

      <InlineErrorBoundary label="inline-renderer">
        <InlineRenderer />
      </InlineErrorBoundary>

      {chat.some(m => m.requiresConfirmation) && !isStreaming && (
        <div className="flex items-center gap-2 px-4 py-2 border-t border-border">
          <button onClick={() => onConfirmAction(true)} className="px-3 py-1 bg-success/10 text-success border border-success/30 rounded text-xs font-bold hover:bg-success/20 transition-colors">APPROVA</button>
          <button onClick={() => onConfirmAction(false)} className="px-3 py-1 bg-danger/10 text-danger border border-danger/30 rounded text-xs font-bold hover:bg-danger/20 transition-colors">RIFIUTA</button>
        </div>
      )}

      <TerminalPrompt
        value={input}
        onChange={setInput}
        onSubmit={onSend}
        disabled={isStreaming || !selectedAgent}
        placeholder={selectedAgent ? 'scrivi un messaggio o /comando...' : 'seleziona un agente...'}
      />
    </div>
  )
}