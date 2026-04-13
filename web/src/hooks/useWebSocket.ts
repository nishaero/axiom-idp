import { useEffect, useRef, useState, useCallback } from 'react'
import type { WebSocketStatus, WebSocketMessage } from '@/types/dashboard'

interface UseWebSocketOptions {
  url: string
  autoReconnect?: boolean
  reconnectInterval?: number
  onMessage?: (message: WebSocketMessage) => void
  onConnect?: () => void
  onDisconnect?: () => void
}

const DEFAULT_RECONNECT_INTERVAL = 3000
const MAX_RECONNECT_ATTEMPTS = 5

export function useWebSocket({
  url,
  autoReconnect = true,
  reconnectInterval = DEFAULT_RECONNECT_INTERVAL,
  onMessage,
  onConnect,
  onDisconnect,
}: UseWebSocketOptions): WebSocketStatus {
  const [status, setStatus] = useState<WebSocketStatus>({
    connected: false,
    reconnectAttempts: 0,
  })

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)

  // Clean up on unmount
  useEffect(() => {
    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      wsRef.current = new WebSocket(url)

      wsRef.current.onopen = () => {
        reconnectAttemptsRef.current = 0
        setStatus({
          connected: true,
          reconnectAttempts: 0,
          lastMessageAt: Date.now(),
        })
        onConnect?.()
      }

      wsRef.current.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          setStatus((prev) => ({
            ...prev,
            lastMessageAt: Date.now(),
          }))
          onMessage?.(message)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      wsRef.current.onclose = () => {
        setStatus((prev) => ({
          connected: false,
          reconnectAttempts: prev.reconnectAttempts,
        }))
        onDisconnect?.()

        if (autoReconnect && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttemptsRef.current += 1
          reconnectTimerRef.current = setTimeout(connect, reconnectInterval)
        }
      }

      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error)
        wsRef.current?.close()
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      if (autoReconnect) {
        reconnectTimerRef.current = setTimeout(connect, reconnectInterval)
      }
    }
  }, [url, autoReconnect, reconnectInterval, onConnect, onDisconnect, onMessage])

  // Disconnect
  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    reconnectAttemptsRef.current = 0
  }, [])

  // Send message
  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
      return true
    }
    return false
  }, [])

  // Initialize connection
  useEffect(() => {
    connect()
  }, [connect])

  return {
    connected: status.connected,
    reconnectAttempts: status.reconnectAttempts,
    lastMessageAt: status.lastMessageAt,
    send,
    disconnect,
  }
}

// Hook for specific metric updates
export function useMetricUpdates<T>(
  url: string,
  metricType: string,
  onMetricUpdate: (data: T) => void,
  options?: Partial<UseWebSocketOptions>
) {
  const handleMetricUpdate = useCallback(
    (message: WebSocketMessage) => {
      if (message.type === `metric-${metricType}`) {
        onMetricUpdate(message.data as T)
      }
    },
    [metricType, onMetricUpdate]
  )

  return useWebSocket({
    url,
    onMessage: handleMetricUpdate,
    ...options,
  })
}
