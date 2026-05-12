import React from 'react'
import type { ChatMessage } from '../store/types'

interface CopilotSettingsProps {
  message: ChatMessage | null
  onClose: () => void
}

export const CopilotSettings: React.FC<CopilotSettingsProps> = ({ message, onClose }) => {
  return (
    <div className="w-1/2 border-l border-border bg-background/30 overflow-auto p-4 font-mono text-xs text-text">
      {message ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between border-b border-border pb-2 mb-4">
            <span className="text-textDim uppercase font-bold text-[10px]">Dettagli Messaggio</span>
            <span className="text-textDim text-[10px]">{new Date(message.createdAt * 1000).toLocaleString()}</span>
          </div>
          <div className="text-textDim text-[10px] uppercase font-bold mb-1">Ruolo</div>
          <div className="text-text lowercase">{message.role}</div>
          <div className="text-textDim text-[10px] uppercase font-bold mb-1">Contenuto</div>
          <div className="whitespace-pre-wrap">{message.content}</div>
          {message.toolCall && (
            <div className="mt-4">
              <div className="text-textDim text-[10px] uppercase font-bold mb-1">Tool Call</div>
              <div className="p-2 bg-surface border border-border rounded text-textDim italic">{message.toolCall}</div>
            </div>
          )}
        </div>
      ) : (
        <div className="flex items-center justify-center h-full text-textDim text-xs italic">
          Seleziona un messaggio per vedere i dettagli
        </div>
      )}
    </div>
  )
}
