import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { RealtimeService } from '../websocket'

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(public url: string) {
    // Simulate connection opening
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    }, 10)
  }

  send(_data: string) {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open')
    }
  }

  close(code?: number, reason?: string) {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code, reason, wasClean: true }))
    }
  }

  // Simulate receiving a message
  simulateMessage(data: any) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }))
    }
  }
}

// Mock global WebSocket
global.WebSocket = MockWebSocket as any

describe('RealtimeService', () => {
  let service: RealtimeService

  beforeEach(() => {
    service = new RealtimeService('ws://localhost:8080/ws')
    vi.clearAllMocks()
  })

  afterEach(() => {
    service.disconnect()
  })

  it('should create service instance', () => {
    expect(service).toBeDefined()
    expect(service.isConnected()).toBe(false)
  })

  it('should connect to WebSocket', async () => {
    await service.connect()
    expect(service.isConnected()).toBe(true)
  })

  it('should handle connection status changes', async () => {
    const mockCallback = vi.fn()
    service.onConnectionChange(mockCallback)

    await service.connect()
    expect(mockCallback).toHaveBeenCalledWith(true)

    service.disconnect()
    expect(mockCallback).toHaveBeenCalledWith(false)
  })

  it('should provide connection status details', () => {
    const status = service.getConnectionStatus()
    expect(status).toHaveProperty('connected')
    expect(status).toHaveProperty('connecting')
    expect(status).toHaveProperty('reconnectAttempts')
    expect(status).toHaveProperty('maxReconnectAttempts')
  })

  it('should handle message parsing errors gracefully', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    
    await service.connect()
    
    // Simulate invalid JSON message
    const ws = (service as any).ws as MockWebSocket
    if (ws.onmessage) {
      ws.onmessage(new MessageEvent('message', { data: 'invalid json' }))
    }

    expect(consoleSpy).toHaveBeenCalledWith('Failed to parse WebSocket message:', expect.any(Error))
    consoleSpy.mockRestore()
  })

  it('should remove connection listeners', () => {
    const mockCallback = vi.fn()
    service.onConnectionChange(mockCallback)
    service.removeConnectionListener(mockCallback)

    // Connection change should not trigger callback
    service.connect()
    expect(mockCallback).not.toHaveBeenCalled()
  })
})