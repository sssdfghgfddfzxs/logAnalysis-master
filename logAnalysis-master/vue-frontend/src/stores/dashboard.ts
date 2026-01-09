import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { DashboardStats, LogDisplay, DashboardChartData, LogFilter } from '@/types'
import { apiService } from '@/services/api'

export const useDashboardStore = defineStore('dashboard', () => {
  // State
  const stats = ref<DashboardStats>({
    total_logs: 0,
    anomaly_count: 0,
    anomaly_rate: 0,
    top_sources: [],
    active_services: 0,
    avg_response_time: 0,
    anomaly_trend: [],
    service_distribution: [],
    level_distribution: []
  })

  const recentLogs = ref<LogDisplay[]>([])
  
  const chartData = ref<DashboardChartData>({
    anomalyTrend: { labels: [], datasets: [] },
    serviceDistribution: [],
    levelDistribution: []
  })

  const filters = ref<LogFilter>({
    timeRange: '24h',
    source: 'all',
    anomalyOnly: false
  })

  const loading = ref(false)
  const error = ref<string | null>(null)

  // Actions
  async function fetchDashboardData() {
    loading.value = true
    error.value = null
    
    try {
      const [statsData, logsData, trendsData] = await Promise.all([
        apiService.getDashboardStats(filters.value.timeRange),
        apiService.getAnalysisResults({ ...filters.value, limit: 10 }),
        apiService.getAnomalyTrends(filters.value.timeRange)
      ])

      stats.value = statsData
      recentLogs.value = logsData.results
      
      // Update chart data
      chartData.value = {
        anomalyTrend: trendsData,
        serviceDistribution: statsData.service_distribution || [],
        levelDistribution: statsData.level_distribution || []
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch dashboard data'
      console.error('Failed to fetch dashboard data:', err)
    } finally {
      loading.value = false
    }
  }

  function updateFilters(newFilters: Partial<LogFilter>) {
    filters.value = { ...filters.value, ...newFilters }
    fetchDashboardData()
  }

  function addRecentLog(log: LogDisplay) {
    recentLogs.value.unshift(log)
    // Keep only the most recent 100 logs
    if (recentLogs.value.length > 100) {
      recentLogs.value = recentLogs.value.slice(0, 100)
    }
  }

  function updateStats(newStats: Partial<DashboardStats>) {
    stats.value = { ...stats.value, ...newStats }
  }

  // Computed
  const hasAnomalies = computed(() => stats.value.anomaly_count > 0)
  const anomalyPercentage = computed(() => 
    (stats.value.anomaly_rate * 100).toFixed(2)
  )

  return {
    // State
    stats,
    recentLogs,
    chartData,
    filters,
    loading,
    error,
    
    // Actions
    fetchDashboardData,
    updateFilters,
    addRecentLog,
    updateStats,
    
    // Computed
    hasAnomalies,
    anomalyPercentage
  }
})