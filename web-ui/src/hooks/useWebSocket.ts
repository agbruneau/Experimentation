import { useEffect, useRef, useCallback } from 'react'
import { useSimulationStore } from '../store/simulationStore'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8083/ws'
const RECONNECT_DELAY = 3000

interface WebSocketMessage {
  type: string
  topic?: string
  data?: unknown
  timestamp: string
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const { setConnectionStatus, addEvent } = useSimulationStore()

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setConnectionStatus('connecting')

    try {
      const ws = new WebSocket(WS_URL)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('WebSocket connected')
        setConnectionStatus('connected')

        // Subscribe to all events
        ws.send(JSON.stringify({ action: 'subscribe', topic: '*' }))
      }

      ws.onmessage = (event) => {
        try {
          const messages = event.data.split('\n')
          messages.forEach((msgStr: string) => {
            if (!msgStr.trim()) return
            const message: WebSocketMessage = JSON.parse(msgStr)
            handleMessage(message)
          })
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setConnectionStatus('disconnected')
        scheduleReconnect()
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        setConnectionStatus('disconnected')
      }
    } catch (err) {
      console.error('Failed to connect WebSocket:', err)
      setConnectionStatus('disconnected')
      scheduleReconnect()
    }
  }, [setConnectionStatus])

  const handleMessage = (message: WebSocketMessage) => {
    switch (message.type) {
      case 'connected':
        console.log('WebSocket welcome:', message.data)
        break

      case 'subscribed':
        console.log('Subscribed to:', message.topic)
        break

      case 'event':
        if (message.topic && message.data) {
          addEvent({
            id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
            type: (message.data as { event_type?: string })?.event_type || 'unknown',
            topic: message.topic,
            timestamp: message.timestamp,
            data: message.data as Record<string, unknown>,
          })
        }
        break

      case 'pong':
        // Keep-alive response
        break

      default:
        console.log('Unknown message type:', message.type)
    }
  }

  const scheduleReconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    reconnectTimeoutRef.current = window.setTimeout(() => {
      console.log('Attempting to reconnect...')
      connect()
    }, RECONNECT_DELAY)
  }, [connect])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setConnectionStatus('disconnected')
  }, [setConnectionStatus])

  const send = useCallback((message: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }, [])

  const subscribe = useCallback((topic: string) => {
    send({ action: 'subscribe', topic })
  }, [send])

  const unsubscribe = useCallback((topic: string) => {
    send({ action: 'unsubscribe', topic })
  }, [send])

  useEffect(() => {
    connect()
    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return { connect, disconnect, subscribe, unsubscribe, send }
}
