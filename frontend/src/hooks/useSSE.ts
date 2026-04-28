import { useEffect, useRef, useCallback } from 'react'
import { useStore } from '../store/useStore'

interface ToolStatusPayload {
  tool_id: string
  tool_name: string
  status: 'started' | 'running' | 'completed' | 'failed'
  progress?: number
  result?: unknown
  error?: string
  duration_ms?: number
}

interface NotificationPayload {
  title: string
  message: string
  type: 'info' | 'success' | 'warning' | 'error'
  link?: string
  data?: unknown
}

interface IngestionProgressPayload {
  task_id: string
  task_name: string
  progress: number
  phase: string
  rows_processed: number
  total_rows?: number
}

interface SystemAlertPayload {
  severity: 'critical' | 'warning' | 'info'
  title: string
  description: string
  component: string
}

export interface SSEHandlers {
  onToolStatus?: (data: ToolStatusPayload) => void
  onNotification?: (data: NotificationPayload) => void
  onIngestionProgress?: (data: IngestionProgressPayload) => void
  onSystemAlert?: (data: SystemAlertPayload) => void
  onError?: (error: Event) => void
  onOpen?: () => void
}

const RECONNECT_BASE_DELAY = 1000
const RECONNECT_MAX_DELAY = 30000

export function useSSE(handlers: SSEHandlers = {}) {
  const abortRef = useRef<AbortController | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectDelayRef = useRef(RECONNECT_BASE_DELAY)
  const handlersRef = useRef(handlers)
  const mountedRef = useRef(true)
  const lastEventIdRef = useRef<string | null>(null)

  handlersRef.current = handlers

  const apiKeyRef = useRef('')
  useEffect(() => {
    const unsub = useStore.subscribe((state) => {
      apiKeyRef.current = state.apiKey
    })
    return unsub
  }, [])

  const connect = useCallback(async () => {
    if (abortRef.current) return // already connected/connecting

    const baseUrl = window.location.origin
    const url = new URL('/api/v1/events', baseUrl)

    // Sync lastEventIdRef from module-level tracker (updated by handleSSEEvent)
    if (lastEventIdInternal) {
      lastEventIdRef.current = lastEventIdInternal
    }

    // Last-Event-ID is a forbidden header for fetch() in browsers,
    // so we send it as a query param. The api_key is sent as a header.
    if (lastEventIdRef.current) {
      url.searchParams.set('lastEventId', lastEventIdRef.current)
    }

    const headers: Record<string, string> = {}
    if (apiKeyRef.current) {
      headers['X-Aleph-Api-Key'] = apiKeyRef.current
    }

    const abortController = new AbortController()
    abortRef.current = abortController

    try {
      const response = await fetch(url.toString(), {
        headers,
        signal: abortController.signal,
      })

      if (!response.ok) {
        // Non-200: trigger error reconnection
        abortRef.current = null
        handlersRef.current.onError?.(new Event('error'))
        scheduleReconnect()
        return
      }

      handlersRef.current.onOpen?.()
      reconnectDelayRef.current = RECONNECT_BASE_DELAY

      const reader = response.body?.getReader()
      if (!reader) {
        handlersRef.current.onError?.(new Event('error'))
        scheduleReconnect()
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      // Read the SSE stream
      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const events = extractSSEEvents(buffer)
          buffer = events.remainder

          for (const evt of events.parsed) {
            handleSSEEvent(handlersRef.current, evt)
          }
        }
      } catch (err: unknown) {
        // AbortError = intentional disconnect, don't reconnect
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }
      }
    } catch {
      // Network error: reconnect
    }

    abortRef.current = null

    // Stream ended (server closed or network error): reconnect
    if (mountedRef.current) {
      scheduleReconnect()
    }
  }, [])

  function scheduleReconnect() {
    const delay = Math.min(reconnectDelayRef.current, RECONNECT_MAX_DELAY)
    reconnectDelayRef.current = Math.min(
      reconnectDelayRef.current * 2,
      RECONNECT_MAX_DELAY,
    )
    reconnectTimerRef.current = setTimeout(() => {
      if (mountedRef.current) connect()
    }, delay)
  }

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    const timer = setTimeout(connect, 100)
    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [connect, disconnect])

  return { connect, disconnect }
}

export function useToolStatusSSE() {
  const addToast = useStore((s) => s.addToast)

  useSSE({
    onToolStatus: (data) => {
      if (data.status === 'completed') {
        addToast({
          message: `${data.tool_name} completato${data.duration_ms ? ` in ${(data.duration_ms / 1000).toFixed(1)}s` : ''}`,
          type: 'success',
          context: 'tool-status',
        })
      } else if (data.status === 'failed') {
        addToast({
          message: `${data.tool_name} fallito: ${data.error || 'errore sconosciuto'}`,
          type: 'error',
          context: 'tool-status',
        })
      } else if (data.status === 'started') {
        addToast({
          message: `${data.tool_name} avviato...`,
          type: 'info',
          context: 'tool-status',
        })
      }
    },
  })
}

export function useNotificationSSE() {
  const addToast = useStore((s) => s.addToast)

  useSSE({
    onNotification: (data) => {
      addToast({
        message: data.message,
        type: data.type === 'error' ? 'error' : 'info',
        context: 'notification',
      })
    },
  })
}

// SSE stream parsing utilities

interface ParsedSSEEvent {
  event: string
  data: string
  id: string
}

function extractSSEEvents(buffer: string): { parsed: ParsedSSEEvent[]; remainder: string } {
  const parsed: ParsedSSEEvent[] = []
  const lines = buffer.split('\n')
  let current: Partial<ParsedSSEEvent> = {}
  let remainder = ''
  let i = 0

  while (i < lines.length) {
    const line = lines[i]
    if (line === '') {
      // Empty line = end of event
      if (current.event !== undefined || current.data !== undefined) {
        parsed.push({
          event: current.event || '',
          data: current.data || '',
          id: current.id || '',
        })
        current = {}
      }
      i++
      continue
    }

    if (line.startsWith('event: ')) {
      current.event = line.slice(7)
    } else if (line.startsWith('data: ')) {
      if (current.data) {
        current.data += '\n' + line.slice(6)
      } else {
        current.data = line.slice(6)
      }
    } else if (line.startsWith('id: ')) {
      current.id = line.slice(4)
    } else if (line.startsWith('retry: ')) {
      // retry is informational, skip
    }
    // :keepalive lines have no event field - ignore them
    i++
  }

  // Any uncompleted event data stays in the buffer
  if (current.event !== undefined || current.data !== undefined) {
    remainder = ''
    if (current.event !== undefined) remainder += 'event: ' + current.event + '\n'
    if (current.data !== undefined) remainder += 'data: ' + current.data + '\n'
    if (current.id !== undefined) remainder += 'id: ' + current.id + '\n'
  }

  return { parsed, remainder }
}

function handleSSEEvent(handlers: SSEHandlers, evt: ParsedSSEEvent) {
  if (evt.id) {
    // Track last event ID for reconnection
    lastEventIdInternal = evt.id
  }

  if (!evt.event) return // keepalive or unknown

  try {
    const data = evt.data ? JSON.parse(evt.data) : undefined
    switch (evt.event) {
      case 'tool_status':
        handlers.onToolStatus?.(data)
        break
      case 'notification':
        handlers.onNotification?.(data)
        break
      case 'ingestion_progress':
        handlers.onIngestionProgress?.(data)
        break
      case 'system_alert':
        handlers.onSystemAlert?.(data)
        break
      default:
      // Unknown event type, skip
    }
  } catch {
    // JSON parse error, skip
  }
}

// Module-level last-event-id tracker for reconnection
let lastEventIdInternal: string | null = null
