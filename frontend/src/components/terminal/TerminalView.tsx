import { memo } from 'react'
import { CopilotView } from '../CopilotView'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'

const TerminalViewInner = () => {
  const currentView = useStore(s => s.currentView)
  const showInlinePanel = useStore(s => s.showInlinePanel)
  const agents = useStore(s => s.agents)
  const selectedAgent = useStore(s => s.selectedAgent)
  const setSelectedAgent = useStore(s => s.setSelectedAgent)
  const chat = useStore(s => s.chat)
  const input = useStore(s => s.input)
  const setInput = useStore(s => s.setInput)
  const isStreaming = useStore(s => s.isStreaming)
  const cancelStream = useStore(s => s.cancelStream)
  const clearChat = useStore(s => s.clearChat)
  const { onSend, onConfirmAction } = useAppActions()

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div className="px-4 py-1.5 border-b border-border flex items-center gap-2 select-none">
        <span className="text-xs font-mono text-primary font-bold">aleph-v2</span>
        <span className="text-xs font-mono text-textDim">❯</span>
        <span className="text-xs font-mono text-textMuted">terminal</span>
        <span className="flex-1" />
        <span className="text-[10px] font-mono text-textDim bg-surfaceAlt px-2 py-0.5 rounded">
          {selectedAgent ? `${selectedAgent}` : 'no agent'}
        </span>
      </div>
      <CopilotView
        agents={agents}
        selectedAgent={selectedAgent}
        setSelectedAgent={setSelectedAgent}
        chat={chat}
        input={input}
        setInput={setInput}
        onSend={onSend}
        isStreaming={isStreaming}
        onCancelStream={() => cancelStream()}
        onConfirmAction={onConfirmAction}
        onClearChat={() => clearChat()}
      />
    </div>
  )
}

export const TerminalView = memo(TerminalViewInner)
