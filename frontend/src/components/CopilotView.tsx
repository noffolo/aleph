import React, { useRef, useEffect, useState } from 'react'
import { t } from '../i18n'
import { useStore } from '../store/useStore'
import { TerminalPrompt, TerminalOutput, escapeHtml } from './terminal'
import { InlineRenderer } from './terminal/InlineRenderer'
import { InlineErrorBoundary } from './InlineErrorBoundary'
import { SplitSquareHorizontal } from 'lucide-react'
import { ChatSearchBar } from './ChatSearchBar'
import { ChatExportMenu } from './ChatExportMenu'
import { useSSE, SSEConnectionStatus } from '../hooks/useSSE'
import type { ChatMessage } from '../store/types'

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
const scrollRef = useRef<HTMLDivElement>(null)
const sentinelRef = useRef<HTMLDivElement>(null)
const { splitView, setSplitView, chatSearchQuery, setChatSearchQuery } = useStore()
const [selectedMsgIndex, setSelectedMsgIndex] = useState<number | null>(null)
const [isAtBottom, setIsAtBottom] = useState(true)
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

const commands = [
  { name: 'help', desc: 'Mostra l\'aiuto' },
  { name: 'clear', desc: 'Pulisce la chat' },
  { name: 'settings', desc: 'Apri impostazioni' },
  { name: 'agents', desc: 'Gestisci agenti' },
  { name: 'tools', desc: 'Gestisci strumenti' },
]

const handleCommandSelect = (cmd: string) => {
  setInput(`/${cmd} `)
  setShowCommands(false)
}


  useEffect(() => {
    if (!sentinelRef.current) return

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsAtBottom(entry.isIntersecting)
      },
      { threshold: 1.0 }
    )

    observer.observe(sentinelRef.current)
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    if (isAtBottom) {
      scrollRef.current?.scrollTo(0, scrollRef.current.scrollHeight)
    }
  }, [chat, isStreaming, isAtBottom])

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

  return (
    <div role="region" aria-label="Chat" className="flex flex-col h-full bg-surface rounded-lg border border-border overflow-hidden">
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
<div className="flex items-center gap-4">
<button
onClick={() => setSplitView(!splitView)}
className={`p-1 rounded transition-colors ${splitView ? 'text-primary bg-primary/10' : 'text-textMuted hover:text-text'}`}
title={t('copilot.splitView')}
aria-label={t('copilot.splitView')}
>
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
  <div 
    className={`w-2 h-2 rounded-full transition-colors ${
      status === 'connected' ? 'bg-success shadow-[0_0_4px_#10b981]' : 
      status === 'reconnecting' ? 'bg-yellow-400 shadow-[0_0_4px_#facc15]' : 
      'bg-danger shadow-[0_0_4px_#ef4444]'
    }`} 
    title={`SSE: ${status}`}
  />
  {reconnectCount > 0 && (
    <span className="text-[10px] font-mono text-textDim italic">
      ×{reconnectCount}
    </span>
  )}
</div>
</div>

      </div>

      <ChatSearchBar 
        query={chatSearchQuery} 
        setQuery={setChatSearchQuery} 
        matchCount={filteredChat.length} 
      />

       <div className="flex-1 flex overflow-hidden">
         <div ref={scrollRef} className={`relative flex-1 overflow-auto transition-all duration-200 ${splitView ? 'max-w-1/2' : 'w-full'}`}>
           <TerminalOutput 
             lines={terminalLines} 
             isStreaming={isStreaming} 
             onMessageClick={(id) => setSelectedMsgIndex(id)}
           />
           <div ref={sentinelRef} className="h-px w-full" />
           {!isAtBottom && (
<button
  onClick={() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
    setIsAtBottom(true)
  }}
  className="absolute bottom-4 right-4 w-8 h-8 rounded-full bg-primary text-background flex items-center justify-center shadow-lg hover:bg-primary/80 transition-all z-10"
  title="Torna in fondo"
  aria-label="Scolla verso il basso"
>
  ↓
</button>
           )}
         </div>
         {splitView && (
          <div className="w-1/2 border-l border-border bg-background/30 overflow-auto p-4 font-mono text-xs text-text">
            {selectedMsgIndex !== null && chat[selectedMsgIndex] ? (
              <div className="space-y-4">
                <div className="flex items-center justify-between border-b border-border pb-2 mb-4">
                  <span className="text-textDim uppercase font-bold text-[10px]">Dettagli Messaggio</span>
                  <span className="text-textDim text-[10px]">{new Date(chat[selectedMsgIndex].createdAt * 1000).toLocaleString()}</span>
                </div>
                <div className="text-textDim text-[10px] uppercase font-bold mb-1">Ruolo</div>
                <div className="text-text lowercase">{chat[selectedMsgIndex].role}</div>
                <div className="text-textDim text-[10px] uppercase font-bold mb-1">Contenuto</div>
                <div className="whitespace-pre-wrap">{chat[selectedMsgIndex].content}</div>
                {chat[selectedMsgIndex].toolCall && (
                  <div className="mt-4">
                    <div className="text-textDim text-[10px] uppercase font-bold mb-1">Tool Call</div>
                    <div className="p-2 bg-surface border border-border rounded text-textDim italic">{chat[selectedMsgIndex].toolCall}</div>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-textDim text-xs italic">
                Seleziona un messaggio per vedere i dettagli
              </div>
            )}
          </div>
        )}
      </div>

<InlineErrorBoundary label="inline-renderer">
<InlineRenderer />
</InlineErrorBoundary>

{showCommands && (
<div className="absolute bottom-16 left-4 w-64 bg-surface border border-border rounded-md shadow-xl z-50 overflow-hidden animate-in fade-in slide-in-from-bottom-2 duration-200">
<div className="p-2 text-[10px] font-bold text-textDim uppercase border-b border-border bg-background/50">
Comandi Disponibili
</div>
<div className="max-h-60 overflow-auto p-1">
{commands.filter(c => c.name.toLowerCase().includes(commandInput.toLowerCase())).map(cmd => (
<button
key={cmd.name}
onClick={() => handleCommandSelect(cmd.name)}
className="w-full text-left px-2 py-1.5 rounded hover:bg-primary/10 transition-colors group"
>
<div className="flex items-center justify-between">
<span className="text-xs font-mono text-text group-hover:text-primary">/{cmd.name}</span>
<span className="text-[10px] text-textDim italic">{cmd.desc}</span>
</div>
</button>
))}
{commands.filter(c => c.name.toLowerCase().includes(commandInput.toLowerCase())).length === 0 && (
<div className="p-2 text-xs text-textDim italic text-center">Nessun comando trovato</div>
)}
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
          <div 
            className={`w-2 h-2 rounded-full transition-colors ${
              status === 'connected' ? 'bg-success shadow-[0_0_4px_#10b981]' : 
              status === 'reconnecting' ? 'bg-yellow-400 shadow-[0_0_4px_#facc15]' : 
              'bg-danger shadow-[0_0_4px_#ef4444]'
            }`} 
            title={`SSE Status: ${status}`}
          />
          <span className="text-[10px] font-mono text-textDim uppercase tracking-wider">
            SSE: {status}
          </span>
          {reconnectCount > 0 && (
            <span className="text-[10px] font-mono text-textDim italic ml-1">
              (×{reconnectCount})
            </span>
          )}
        </div>
        <TerminalPrompt
          aria-label="Message input"
          value={input}
         onChange={setInput}
         onSubmit={onSend}
         disabled={isStreaming || !selectedAgent}
         placeholder={selectedAgent ? 'scrivi un messaggio o /comando...' : 'seleziona un agente...'}
       />


    </div>
  )
});