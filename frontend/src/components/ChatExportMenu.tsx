import React, { useState } from 'react'
import { Download, ChevronDown } from 'lucide-react'
import { ChatMessage } from '../store/types'

interface ChatExportMenuProps {
  messages: ChatMessage[]
}

export const ChatExportMenu: React.FC<ChatExportMenuProps> = ({ messages }) => {
  const [isOpen, setIsOpen] = useState(false)

  const exportJSON = () => {
    const data = JSON.stringify(messages, null, 2)
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `aleph-chat-export-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  const exportCSV = () => {
    const header = 'Role,Content,CreatedAt,ToolCall\n'
    const rows = messages.map(m => {
      const escapedContent = `"${m.content.replace(/"/g, '""')}"`
      const escapedTool = m.toolCall ? `"${m.toolCall.replace(/"/g, '""')}"` : ''
      return `${m.role},${escapedContent},${m.createdAt},${escapedTool}`
    }).join('\n')
    
    const blob = new Blob([header + rows], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `aleph-chat-export-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="relative">
      <button 
        onClick={() => setIsOpen(!isOpen)} 
        className="flex items-center gap-1 text-textMuted hover:text-text transition-colors text-xs font-bold"
      >
        <Download className="w-3 h-3" />
        ESPORTA
        <ChevronDown className={`w-3 h-3 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>
      
      {isOpen && (
        <div className="absolute right-0 mt-1 w-32 bg-surface border border-border rounded shadow-xl z-50 overflow-hidden">
          <button 
            onClick={() => { exportJSON(); setIsOpen(false); }} 
            className="w-full text-left px-3 py-2 text-xs font-mono hover:bg-background transition-colors border-b border-border/50"
          >
            JSON
          </button>
          <button 
            onClick={() => { exportCSV(); setIsOpen(false); }} 
            className="w-full text-left px-3 py-2 text-xs font-mono hover:bg-background transition-colors"
          >
            CSV
          </button>
        </div>
      )}
    </div>
  )
}
