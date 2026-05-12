import React, { useEffect, useState } from 'react'
import { t } from '../i18n'
import { TerminalPrompt, escapeHtml } from './terminal'
import { SplitSquareHorizontal } from 'lucide-react'
import { ChatSearchBar } from './ChatSearchBar'
import { useSSE } from '../hooks/useSSE'
import type { ChatMessage } from '../store/types'
import { CopilotChat } from './CopilotChat'
import { CopilotSettings } from './CopilotSettings'
import { COMMAND_REGISTRY } from '../commands'

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

export const CopilotView: React.FC<CopilotViewProps> = React.memo(({
  agents, selectedAgent, setSelectedAgent,
  chat, input, setInput, onSend, isStreaming, onCancelStream, onConfirmAction, onClearChat
}) => {
  const { status, reconnectCount } = useSSE()
  const [showMessageDetail, setShowMessageDetail] = useState(false)
  const [chatSearchQuery, setChatSearchQuery] = useState('')
  const [selectedMsgIndex, setSelectedMsgIndex] = useState<number | null>(null)
  const [showCommands, setShowCommands] = useState(false)
  const [commandInput, setCommandInput] = useState('')

  useEffect(() => {
    if (input.startsWith('/')) {
      setCommandInput(input.slice(1))
      setShowCommands(true)
    } else {
      setShowCommands(false)
    }
  }, [input])

  const handleCommandSelect = (cmd: string) => {
    setInput(`${cmd} `)
    setShowCommands(false)
  }

  const filteredChat = chat.filter(msg =>
    msg.content.toLowerCase().includes(chatSearchQuery.toLowerCase()) ||
    (msg.toolCall && msg.toolCall.toLowerCase().includes(chatSearchQuery.toLowerCase()))
  )

  const terminalLines = filteredChat.map((msg, i) => ({
    id: i,
    type: msg.role === 'user' ? 'input' as const : msg.toolCall ? 'tool' as const : 'output' as const,
    content: escapeHtml(msg.content || msg.toolCall || ''),
    timestamp: msg.createdAt,
  }))

  const sseDotClass = `w-2 h-2 rounded-full transition-colors ${
    status === 'connected' ? 'bg-success shadow-[0_0_4px_#10b981]' :
    status === 'reconnecting' ? 'bg-yellow-400 shadow-[0_0_4px_#facc15]' :
    'bg-danger shadow-[0_0_4px_#ef4444]'
  }`

  return (
    <div role="region" aria-label="Chat" className="flex flex-col h-full bg-surface rounded-lg border border-border overflow-hidden">
      <div className="h-9 flex items-center justify-between px-4 border-b border-border shrink-0">
        <div className="flex items-center gap-2">
          <span className="text-primary text-xs font-bold">COPILOT</span>
          <span className="text-textDim text-xs">│</span>
          <select value={selectedAgent} onChange={(e) => setSelectedAgent(e.target.value)} disabled={isStreaming}
            className="bg-transparent text-text text-xs outline-none cursor-pointer disabled:opacity-50">
            <option value="" className="bg-surface text-text">seleziona agente...</option>
            {agents.map(a => (
              <option key={a.id} value={a.id} className="bg-surface text-text">{a.name} ({a.model})</option>
            ))}
          </select>
        </div>
        <div className="flex items-center gap-4">
          <button onClick={() => setShowMessageDetail(!showMessageDetail)}
            className={`p-1 rounded transition-colors ${showMessageDetail ? 'text-primary bg-primary/10' : 'text-textMuted hover:text-text'}`}
            title={t('copilot.showMessageDetail')} aria-label={t('copilot.showMessageDetail')}>
            <SplitSquareHorizontal className="w-3 h-3" />
          </button>
          {chat.length > 0 && (
            <div className="flex items-center gap-4">
              {isStreaming && (
                <button onClick={onCancelStream} className="text-danger hover:text-danger-bright text-xs font-bold transition-colors" title={t('copilot.cancelStream')}>
                  ⏹ STOP
                </button>
              )}
              <button onClick={onClearChat} className="text-textMuted hover:text-text transition-colors text-xs font-bold" title={t('copilot.clearChat')}>
                PULISCI
              </button>
            </div>
          )}
          <div className="flex items-center gap-2 ml-auto">
            <div className={sseDotClass} title={`SSE: ${status}`} />
            {reconnectCount > 0 && (
              <span className="text-[10px] font-mono text-textDim italic">×{reconnectCount}</span>
            )}
          </div>
        </div>
      </div>

      <ChatSearchBar query={chatSearchQuery} setQuery={setChatSearchQuery} matchCount={filteredChat.length} />

      <div className="flex-1 flex overflow-hidden">
        <CopilotChat lines={terminalLines} isStreaming={isStreaming}
          onMessageClick={(id) => setSelectedMsgIndex(id)} />
        {showMessageDetail && (
          <CopilotSettings
            message={selectedMsgIndex !== null ? chat[selectedMsgIndex] : null}
            onClose={() => setShowMessageDetail(false)} />
        )}
      </div>

      {showCommands && (
        <div className="absolute bottom-16 left-4 w-64 bg-surface border border-border rounded-md shadow-xl z-50 overflow-hidden animate-in fade-in slide-in-from-bottom-2 duration-200">
          <div className="p-2 text-[10px] font-bold text-textDim uppercase border-b border-border bg-background/50">
            Comandi Disponibili
          </div>
          <div className="max-h-60 overflow-auto p-1">
            {(() => {
              const relevantCommands = COMMAND_REGISTRY.filter(c =>
                ['/help', '/clear', '/settings', '/agents', '/tools'].includes(c.name)
              ).filter(c => c.name.slice(1).includes(commandInput.toLowerCase()))
              if (relevantCommands.length === 0) {
                return <div className="p-2 text-xs text-textDim italic text-center">Nessun comando trovato</div>
              }
              return relevantCommands.map(cmd => (
                <button key={cmd.name} onClick={() => handleCommandSelect(cmd.name)}
                  className="w-full text-left px-2 py-1.5 rounded hover:bg-primary/10 transition-colors group">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono text-text group-hover:text-primary">{cmd.name}</span>
                    <span className="text-[10px] text-textDim italic">{cmd.description}</span>
                  </div>
                </button>
              ))
            })()}
          </div>
        </div>
      )}

      {chat.some(m => m.requiresConfirmation) && !isStreaming && (
        <div className="flex items-center gap-2 px-4 py-2 border-t border-border">
          <button onClick={() => onConfirmAction(true)} className="px-3 py-1 bg-success/10 text-success border border-success/30 rounded text-xs font-bold hover:bg-success/20 transition-colors">APPROVA</button>
          <button onClick={() => onConfirmAction(false)} className="px-3 py-1 bg-danger/10 text-danger border border-danger/30 rounded text-xs font-bold hover:bg-danger/20 transition-colors">RIFIUTA</button>
        </div>
      )}

      <div className="flex items-center gap-2 px-4 py-1 border-t border-border bg-surface/30">
        <div className={sseDotClass} title={`SSE Status: ${status}`} />
        <span className="text-[10px] font-mono text-textDim uppercase tracking-wider">SSE: {status}</span>
        {reconnectCount > 0 && (
          <span className="text-[10px] font-mono text-textDim italic ml-1">(×{reconnectCount})</span>
        )}
      </div>

      <TerminalPrompt aria-label="Message input" value={input} onChange={setInput} onSubmit={onSend}
        disabled={isStreaming || !selectedAgent}
        placeholder={selectedAgent ? 'scrivi un messaggio o /comando...' : 'seleziona un agente...'} />
    </div>
  )
})
