import type { RealtimeMessage, LogDisplay, DashboardStats } from '@/types'
import { useDashboardStore } from '@/stores/dashboard'

export class RealtimeService {
  private ws: WebSocket | null = null
  private wsUrl: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private isConnecting = false
  private heartbeatInterval: number | null = null
  private connectionListeners: Array<(connected: boolean) => void> = []

  constructor(wsUrl?: string) {
    // 在生产环境中使用相对路径，开发环境使用代理路径
    if (wsUrl) {
      this.wsUrl = wsUrl
    } else if (import.meta.env.PROD) {
      // 生产环境使用相对路径，通过nginx代理
      this.wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
    } else {
      // 开发环境使用Vite代理路径
      this.wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
    }
  }

  // Add connection status listener
  onConnectionChange(callback: (connected: boolean) => void) {
    this.connectionListeners.push(callback)
  }

  // Remove connection status listener
  removeConnectionListener(callback: (connected: boolean) => void) {
    const index = this.connectionListeners.indexOf(callback)
    if (index > -1) {
      this.connectionListeners.splice(index, 1)
    }
  }

  // Notify all listeners about connection status change
  private notifyConnectionChange(connected: boolean) {
    this.connectionListeners.forEach(callback => {
      try {
        callback(connected)
      } catch (error) {
        console.error('Error in connection listener:', error)
      }
    })
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
        resolve()
        return
      }

      this.isConnecting = true

      try {
        this.ws = new WebSocket(this.wsUrl)

        this.ws.onopen = () => {
          console.log('WebSocket connected')
          this.reconnectAttempts = 0
          this.isConnecting = false
          this.startHeartbeat()
          this.notifyConnectionChange(true)
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const message: RealtimeMessage = JSON.parse(event.data)
            this.handleRealtimeUpdate(message)
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error)
          }
        }

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason)
          this.isConnecting = false
          this.ws = null
          this.stopHeartbeat()
          this.notifyConnectionChange(false)
          
          if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.attemptReconnect()
          }
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          this.isConnecting = false
          this.notifyConnectionChange(false)
          reject(error)
        }

      } catch (error) {
        this.isConnecting = false
        this.notifyConnectionChange(false)
        reject(error)
      }
    })
  }

  private handleRealtimeUpdate(message: RealtimeMessage) {
    const dashboardStore = useDashboardStore()

    switch (message.type) {
      case 'new_anomaly':
        if (message.data.log) {
          dashboardStore.addRecentLog(message.data.log as LogDisplay)
          dashboardStore.updateStats({ 
            anomaly_count: dashboardStore.stats.anomaly_count + 1 
          })
          this.showNotification('检测到新异常', message.data.log.message, 'warning')
        }
        break

      case 'stats_update':
        if (message.data.stats) {
          dashboardStore.updateStats(message.data.stats as Partial<DashboardStats>)
        }
        break

      case 'system_alert':
        this.showNotification('系统告警', message.data.message, 'error')
        break

      case 'connection_established':
        console.log('WebSocket connection established')
        break

      case 'pong':
        // Heartbeat response, no action needed
        break

      default:
        console.log('Unknown message type:', message.type)
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      this.showNotification('连接失败', '无法连接到服务器，请刷新页面重试', 'error')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)

    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`)
    this.showNotification('连接断开', `正在尝试重新连接... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`, 'warning')

    setTimeout(() => {
      this.connect().catch(error => {
        console.error('Reconnection failed:', error)
      })
    }, delay)
  }

  // Heartbeat mechanism to keep connection alive
  private startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatInterval = window.setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        try {
          this.ws.send(JSON.stringify({ type: 'ping' }))
        } catch (error) {
          console.error('Failed to send heartbeat:', error)
        }
      }
    }, 30000) // Send heartbeat every 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval !== null) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  private showNotification(title: string, message: string, type: 'success' | 'warning' | 'error' | 'info' = 'info') {
    // Use Element Plus notification if available
    if (typeof ElNotification !== 'undefined') {
      ElNotification({
        title,
        message,
        type,
        duration: type === 'error' ? 0 : 5000, // Error notifications don't auto-close
        position: 'top-right'
      })
    } else {
      // Fallback to console log
      console.log(`[${type.toUpperCase()}] ${title}: ${message}`)
    }
  }

  disconnect() {
    this.stopHeartbeat()
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
    this.notifyConnectionChange(false)
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }

  // Get connection status details
  getConnectionStatus() {
    return {
      connected: this.isConnected(),
      connecting: this.isConnecting,
      reconnectAttempts: this.reconnectAttempts,
      maxReconnectAttempts: this.maxReconnectAttempts
    }
  }

  // Force reconnection
  forceReconnect() {
    this.disconnect()
    this.reconnectAttempts = 0
    return this.connect()
  }
}

// Create singleton instance
export const realtimeService = new RealtimeService()