import { CopilotView } from '../CopilotView'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'

export function TerminalView() {
  const store = useStore()
  const { onSend, onConfirmAction } = useAppActions()

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div className="px-4 py-1.5 border-b border-border flex items-center gap-2 select-none">
        <span className="text-xs font-mono text-primary font-bold">aleph-v2</span>
        <span className="text-xs font-mono text-textDim">❯</span>
        <span className="text-xs font-mono text-textMuted">terminal</span>
        <span className="flex-1" />
        <span className="text-[10px] font-mono text-textDim bg-surfaceAlt px-2 py-0.5 rounded">
          {store.selectedAgent ? `${store.selectedAgent}` : 'no agent'}
        </span>
      </div>
      <CopilotView
        agents={store.agents}
        selectedAgent={store.selectedAgent}
        setSelectedAgent={store.setSelectedAgent}
        chat={store.chat}
        input={store.input}
        setInput={store.setInput}
        onSend={onSend}
        isStreaming={store.isStreaming}
        onCancelStream={() => store.cancelStream()}
        onConfirmAction={onConfirmAction}
        onClearChat={() => store.clearChat()}
      />
    </div>
  )
}
