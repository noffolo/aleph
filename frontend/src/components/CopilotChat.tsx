import React, { useRef, useEffect, useState } from 'react'
import { TerminalOutput } from './terminal'
import type { TerminalLine } from './terminal/TerminalOutput'

interface CopilotChatProps {
  lines: TerminalLine[]
  isStreaming: boolean
  onMessageClick?: (id: number) => void
}

export const CopilotChat: React.FC<CopilotChatProps> = ({ lines, isStreaming, onMessageClick }) => {
  const [isAtBottom, setIsAtBottom] = useState(true)
  const scrollRef = useRef<HTMLDivElement>(null)
  const sentinelRef = useRef<HTMLDivElement>(null)

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
  }, [lines, isStreaming, isAtBottom])

  const scrollToBottom = () => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
    setIsAtBottom(true)
  }

  return (
    <div ref={scrollRef} className="relative flex-1 overflow-auto">
      <TerminalOutput lines={lines} isStreaming={isStreaming} onMessageClick={onMessageClick} />
      <div ref={sentinelRef} className="h-px w-full" />
      {!isAtBottom && (
        <button
          onClick={scrollToBottom}
          className="absolute bottom-4 right-4 w-8 h-8 rounded-full bg-primary text-background flex items-center justify-center shadow-lg hover:bg-primary/80 transition-all z-10"
          title="Torna in fondo"
          aria-label="Scolla verso il basso"
        >
          ↓
        </button>
      )}
    </div>
  )
}
