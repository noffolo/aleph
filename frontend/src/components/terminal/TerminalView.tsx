import { memo, useState } from 'react'
import { CopilotView } from '../CopilotView'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'

const TerminalViewInner = () => {
  const agents = useStore(s => s.agents)
  const selectedAgent = useStore(s => s.selectedAgent)
  const setSelectedAgent = useStore(s => s.setSelectedAgent)
  const messages = useStore(s => s.messages)
  const isStreaming = useStore(s => s.isStreaming)
  const clearMessages = useStore(s => s.clearMessages)
  const [input, setInput] = useState('')
  const { onSend: onSendAction, onConfirmAction, onCancelStream } = useAppActions()
  const handleSend = () => {
    if (input.trim()) {
      onSendAction(input)
      setInput('')
    }
  }

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
        chat={messages}
        input={input}
        setInput={setInput}
        onSend={handleSend}
        isStreaming={isStreaming}
        onCancelStream={onCancelStream}
        onConfirmAction={onConfirmAction}
        onClearChat={() => clearMessages()}
      />
    </div>
  )
}

export const TerminalView = memo(TerminalViewInner)
