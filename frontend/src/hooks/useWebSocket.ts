import { useCallback, useEffect, useRef, useState } from 'react'

import type { WebSocketEventMessage } from '../types'

type WebSocketMessageHandler = (message: WebSocketEventMessage) => void

interface UseWebSocketOptions {
  token?: string | null
  enabled?: boolean
  heartbeatInterval?: number
  baseReconnectDelay?: number
  maxReconnectDelay?: number
  onMessage?: WebSocketMessageHandler
  onConnectedChange?: (connected: boolean) => void
  handlers?: Partial<Record<string, WebSocketMessageHandler>>
}

interface UseWebSocketResult {
  connected: boolean
  send: (payload: string | Record<string, unknown>) => boolean
  disconnect: () => void
  reconnect: () => void
}

const buildWebSocketURL = (token: string): string => {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}/ws?token=${encodeURIComponent(token)}`
}

const parseMessage = (raw: string): WebSocketEventMessage | null => {
  try {
    const parsed = JSON.parse(raw) as WebSocketEventMessage
    if (!parsed || typeof parsed !== 'object' || typeof parsed.type !== 'string') {
      return null
    }
    return parsed
  } catch (_error) {
    return null
  }
}

export const useWebSocket = ({
  token,
  enabled = true,
  heartbeatInterval = 25_000,
  baseReconnectDelay = 1_000,
  maxReconnectDelay = 30_000,
  onMessage,
  onConnectedChange,
  handlers,
}: UseWebSocketOptions): UseWebSocketResult => {
  const [connected, setConnected] = useState<boolean>(false)

  const socketRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)
  const heartbeatTimerRef = useRef<number | null>(null)
  const reconnectAttemptRef = useRef<number>(0)
  const manualCloseRef = useRef<boolean>(false)
  const tokenRef = useRef<string | null>(token ?? null)
  const enabledRef = useRef<boolean>(Boolean(enabled))
  const connectRef = useRef<() => void>(() => undefined)

  const callbacksRef = useRef<{
    onMessage?: WebSocketMessageHandler
    onConnectedChange?: (next: boolean) => void
    handlers?: Partial<Record<string, WebSocketMessageHandler>>
  }>({
    onMessage,
    onConnectedChange,
    handlers,
  })

  useEffect(() => {
    callbacksRef.current = {
      onMessage,
      onConnectedChange,
      handlers,
    }
  }, [handlers, onConnectedChange, onMessage])

  const updateConnected = useCallback((next: boolean): void => {
    setConnected(next)
    callbacksRef.current.onConnectedChange?.(next)
  }, [])

  const clearReconnectTimer = useCallback((): void => {
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
  }, [])

  const stopHeartbeat = useCallback((): void => {
    if (heartbeatTimerRef.current !== null) {
      window.clearInterval(heartbeatTimerRef.current)
      heartbeatTimerRef.current = null
    }
  }, [])

  const closeSocket = useCallback((): void => {
    const socket = socketRef.current
    if (!socket) {
      return
    }

    if (
      socket.readyState === WebSocket.OPEN ||
      socket.readyState === WebSocket.CONNECTING
    ) {
      socket.close(1000, 'closed')
    }

    socketRef.current = null
  }, [])

  const scheduleReconnect = useCallback((): void => {
    if (manualCloseRef.current || !enabledRef.current || !tokenRef.current) {
      return
    }

    if (reconnectTimerRef.current !== null) {
      return
    }

    const delay = Math.min(
      baseReconnectDelay * 2 ** reconnectAttemptRef.current,
      maxReconnectDelay,
    )
    reconnectAttemptRef.current += 1

    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectTimerRef.current = null
      connectRef.current()
    }, delay)
  }, [baseReconnectDelay, maxReconnectDelay])

  const startHeartbeat = useCallback((): void => {
    stopHeartbeat()

    heartbeatTimerRef.current = window.setInterval(() => {
      const socket = socketRef.current
      if (!socket || socket.readyState !== WebSocket.OPEN) {
        return
      }
      socket.send(
        JSON.stringify({
          type: 'ping',
          timestamp: new Date().toISOString(),
        }),
      )
    }, heartbeatInterval)
  }, [heartbeatInterval, stopHeartbeat])

  const connect = useCallback((): void => {
    if (!enabledRef.current || !tokenRef.current || manualCloseRef.current) {
      return
    }

    const currentSocket = socketRef.current
    if (
      currentSocket &&
      (currentSocket.readyState === WebSocket.OPEN ||
        currentSocket.readyState === WebSocket.CONNECTING)
    ) {
      return
    }

    const socket = new WebSocket(buildWebSocketURL(tokenRef.current))
    socketRef.current = socket

    socket.onopen = () => {
      reconnectAttemptRef.current = 0
      clearReconnectTimer()
      updateConnected(true)
      startHeartbeat()
    }

    socket.onmessage = (event) => {
      if (typeof event.data !== 'string') {
        return
      }

      if (event.data === 'pong') {
        return
      }

      const message = parseMessage(event.data)
      if (!message) {
        return
      }

      callbacksRef.current.onMessage?.(message)
      callbacksRef.current.handlers?.[message.type]?.(message)
    }

    socket.onclose = () => {
      socketRef.current = null
      stopHeartbeat()
      updateConnected(false)
      scheduleReconnect()
    }

    socket.onerror = () => {
      updateConnected(false)
    }
  }, [clearReconnectTimer, scheduleReconnect, startHeartbeat, stopHeartbeat, updateConnected])

  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  const disconnect = useCallback((): void => {
    manualCloseRef.current = true
    clearReconnectTimer()
    stopHeartbeat()
    closeSocket()
    updateConnected(false)
  }, [clearReconnectTimer, closeSocket, stopHeartbeat, updateConnected])

  const reconnect = useCallback((): void => {
    manualCloseRef.current = true
    clearReconnectTimer()
    stopHeartbeat()
    closeSocket()
    reconnectAttemptRef.current = 0
    manualCloseRef.current = false
    connectRef.current()
  }, [clearReconnectTimer, closeSocket, stopHeartbeat])

  useEffect(() => {
    tokenRef.current = token ?? null
    enabledRef.current = Boolean(enabled)

    if (!enabled || !token) {
      disconnect()
      return
    }

    manualCloseRef.current = false
    reconnectAttemptRef.current = 0
    connectRef.current()

    return () => {
      manualCloseRef.current = true
      clearReconnectTimer()
      stopHeartbeat()
      closeSocket()
      updateConnected(false)
    }
  }, [clearReconnectTimer, closeSocket, connect, disconnect, enabled, stopHeartbeat, token, updateConnected])

  const send = useCallback((payload: string | Record<string, unknown>): boolean => {
    const socket = socketRef.current
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return false
    }

    const content =
      typeof payload === 'string' ? payload : JSON.stringify(payload)
    socket.send(content)
    return true
  }, [])

  return {
    connected,
    send,
    disconnect,
    reconnect,
  }
}

export type { UseWebSocketOptions, UseWebSocketResult, WebSocketMessageHandler }
