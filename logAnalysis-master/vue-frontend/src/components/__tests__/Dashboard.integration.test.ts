import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDashboardStore } from '../../stores/dashboard'

// Mock the API service
vi.mock('../../services/api', () => ({
  apiService: {
    getDashboardStats: vi.fn().mockResolvedValue({
      total_logs: 1000,
      anomaly_count: 50,
      anomaly_rate: 0.05,
      top_sources: [{ source: 'service-a', count: 100 }, { source: 'service-b', count: 80 }],
      active_services: 3,
      avg_response_time: 150,
      anomaly_trend: [],
      service_distribution: [],
      level_distribution: []
    }),
    getAnalysisResults: vi.fn().mockResolvedValue({
      results: [
        {
          id: '1',
          timestamp: '2024-01-01T10:00:00Z',
          level: 'ERROR',
          message: 'Test error message',
          source: 'test-service',
          isAnomaly: true,
          anomalyScore: 0.85
        }
      ],
      total: 1,
      page: 1,
      pageSize: 20
    }),
    getAnomalyTrends: vi.fn().mockResolvedValue({
      anomalyTrend: [{ time: '2024-01-01T10:00:00Z', count: 5 }],
      serviceDistribution: [{ name: 'service-a', value: 60 }],
      levelDistribution: [{ level: 'ERROR', count: 10 }]
    })
  }
}))

describe('Dashboard Store Real-time Integration', () => {
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('should handle real-time log updates through store', async () => {
    const store = useDashboardStore()
    
    // Simulate adding a new log through real-time update
    const newLog = {
      id: '2',
      timestamp: '2024-01-01T11:00:00Z',
      level: 'WARN' as const,
      message: 'New warning message',
      source: 'new-service',
      isAnomaly: false,
      anomalyScore: 0.2
    }

    store.addRecentLog(newLog)
    
    expect(store.recentLogs).toHaveLength(1)
    expect(store.recentLogs[0]).toEqual(newLog)
  })

  it('should update stats through real-time updates', async () => {
    const store = useDashboardStore()
    
    // Initial stats should be default values
    expect(store.stats.anomaly_count).toBe(0)
    
    // Simulate stats update from WebSocket
    store.updateStats({ anomaly_count: 51, total_logs: 1000 })
    
    expect(store.stats.anomaly_count).toBe(51)
    expect(store.stats.total_logs).toBe(1000)
  })

  it('should limit recent logs to prevent memory issues', async () => {
    const store = useDashboardStore()
    
    // Add 105 logs (more than the 100 limit)
    for (let i = 0; i < 105; i++) {
      store.addRecentLog({
        id: `log-${i}`,
        timestamp: `2024-01-01T${String(i % 24).padStart(2, '0')}:00:00Z`,
        level: 'INFO' as const,
        message: `Log message ${i}`,
        source: 'test-service',
        isAnomaly: false
      })
    }
    
    // Should be limited to 100 logs
    expect(store.recentLogs).toHaveLength(100)
    // Most recent log should be first
    expect(store.recentLogs[0].id).toBe('log-104')
  })

  it('should fetch dashboard data correctly', async () => {
    const store = useDashboardStore()
    
    await store.fetchDashboardData()
    
    // Verify data was fetched and stored
    expect(store.stats.total_logs).toBe(1000)
    expect(store.stats.anomaly_count).toBe(50)
    expect(store.recentLogs).toHaveLength(1)
    expect(store.recentLogs[0].id).toBe('1')
  })
})