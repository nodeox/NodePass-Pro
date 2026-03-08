import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'

import { useWebSocket } from './useWebSocket'
import type { WebSocketEventMessage } from '../types'

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: MockWebSocket[] = []

  url: string
  protocols: string[]
  readyState = MockWebSocket.CONNECTING
  sentData: string[] = []

  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null

  constructor(url: string, protocols?: string | string[]) {
    this.url = url
    this.protocols = Array.isArray(protocols)
      ? protocols
      : protocols
        ? [protocols]
        : []
    MockWebSocket.instances.push(this)
  }

  static reset(): void {
    MockWebSocket.instances = []
  }

  send(data: string): void {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('socket is not open')
    }
    this.sentData.push(data)
  }

  close(code?: number, reason?: string): void {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close', { code: code ?? 1000, reason: reason ?? '' }))
  }

  open(): void {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.(new Event('open'))
  }

  emitMessage(payload: WebSocketEventMessage): void {
    this.onmessage?.(
      new MessageEvent('message', {
        data: JSON.stringify(payload),
      }),
    )
  }

  emitClose(): void {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close', { code: 1006, reason: 'closed by peer' }))
  }
}

describe('useWebSocket', () => {
  const originalWebSocket = global.WebSocket

  beforeEach(() => {
    vi.useFakeTimers()
    MockWebSocket.reset()
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket
  })

  afterEach(() => {
    vi.useRealTimers()
    global.WebSocket = originalWebSocket
    MockWebSocket.reset()
  })

  it('未提供 token 时不建立连接', () => {
    renderHook(() => useWebSocket({ token: null }))
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  it('使用 subprotocol 传递 bearer token 建立连接', async () => {
    const { result } = renderHook(() => useWebSocket({ token: 'token-123' }))

    expect(MockWebSocket.instances).toHaveLength(1)
    const socket = MockWebSocket.instances[0]
    expect(socket.protocols).toEqual(['bearer', 'token-123'])

    act(() => {
      socket.open()
    })

    expect(result.current.connected).toBe(true)
  })

  it('连接成功后可发送 JSON 消息', async () => {
    const { result } = renderHook(() => useWebSocket({ token: 'token-123' }))
    const socket = MockWebSocket.instances[0]

    act(() => {
      socket.open()
    })

    expect(result.current.connected).toBe(true)

    let sent = false
    act(() => {
      sent = result.current.send({ type: 'ping', payload: 'ok' })
    })

    expect(sent).toBe(true)
    expect(socket.sentData).toHaveLength(1)
    expect(socket.sentData[0]).toContain('"type":"ping"')
  })

  it('收到事件后调用 onMessage 和 handlers', async () => {
    const onMessage = vi.fn()
    const onNotice = vi.fn()

    renderHook(() =>
      useWebSocket({
        token: 'token-123',
        onMessage,
        handlers: {
          notice: onNotice,
        },
      }),
    )

    const socket = MockWebSocket.instances[0]

    act(() => {
      socket.open()
      socket.emitMessage({
        type: 'notice',
        payload: {
          id: 1,
        },
      } as unknown as WebSocketEventMessage)
    })

    expect(onMessage).toHaveBeenCalledTimes(1)
    expect(onNotice).toHaveBeenCalledTimes(1)
  })

  it('断开后按退避策略重连', async () => {
    renderHook(() =>
      useWebSocket({
        token: 'token-123',
        baseReconnectDelay: 100,
        maxReconnectDelay: 100,
      }),
    )

    const first = MockWebSocket.instances[0]

    act(() => {
      first.open()
      first.emitClose()
    })

    expect(MockWebSocket.instances).toHaveLength(1)

    await act(async () => {
      vi.advanceTimersByTime(100)
    })

    expect(MockWebSocket.instances).toHaveLength(2)
    expect(MockWebSocket.instances[1].protocols).toEqual(['bearer', 'token-123'])
  })
})
