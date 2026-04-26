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
  const eventSourceRef = useRef<EventSource | null>(null)
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

  const connect = useCallback(() => {
    if (eventSourceRef.current?.readyState === EventSource.OPEN) return

    const baseUrl = window.location.origin
    const url = new URL('/api/v1/events', baseUrl)

    if (apiKeyRef.current) {
      url.searchParams.set('api_key', apiKeyRef.current)
    }
    if (lastEventIdRef.current) {
      url.searchParams.set('lastEventId', lastEventIdRef.current)
    }

    const es = new EventSource(url.toString())
    eventSourceRef.current = es

    es.onopen = () => {
      reconnectDelayRef.current = RECONNECT_BASE_DELAY
      handlersRef.current.onOpen?.()
    }

    es.addEventListener('tool_status', (event: MessageEvent) => {
      try {
        handlersRef.current.onToolStatus?.(JSON.parse(event.data))
      } catch { /**/ }
    })

    es.addEventListener('notification', (event: MessageEvent) => {
      try {
        handlersRef.current.onNotification?.(JSON.parse(event.data))
      } catch { /**/ }
    })

    es.addEventListener('ingestion_progress', (event: MessageEvent) => {
      try {
        handlersRef.current.onIngestionProgress?.(JSON.parse(event.data))
      } catch { /**/ }
    })

    es.addEventListener('system_alert', (event: MessageEvent) => {
      try {
        handlersRef.current.onSystemAlert?.(JSON.parse(event.data))
      } catch { /**/ }
    })

    es.onerror = (event: Event) => {
      handlersRef.current.onError?.(event)
      es.close()
      eventSourceRef.current = null

      if (mountedRef.current) {
        const delay = Math.min(reconnectDelayRef.current, RECONNECT_MAX_DELAY)
        reconnectDelayRef.current = Math.min(
          reconnectDelayRef.current * 2,
          RECONNECT_MAX_DELAY,
        )
        reconnectTimerRef.current = setTimeout(() => {
          if (mountedRef.current) connect()
        }, delay)
      }
    }
  }, [])

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
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
