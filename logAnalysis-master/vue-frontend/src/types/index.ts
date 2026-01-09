// Dashboard statistics interface
export interface DashboardStats {
  total_logs: number
  anomaly_count: number
  anomaly_rate: number
  top_sources: Array<{ source: string; count: number }>
  active_services: number
  avg_response_time: number
  anomaly_trend: Array<{ time: string; count: number }>
  service_distribution: Array<{ name: string; value: number }>
  level_distribution: Array<{ level: string; count: number }>
}

// Chart data structures
export interface ChartData {
  labels: string[]
  datasets: Array<{
    label: string
    data: number[]
    borderColor: string
    backgroundColor: string
  }>
}

// Dashboard chart data (separate from individual chart data)
export interface DashboardChartData {
  anomalyTrend: ChartData
  serviceDistribution: Array<{ name: string; value: number }>
  levelDistribution: Array<{ level: string; count: number }>
}

// Log display model
export interface LogDisplay {
  id: string
  timestamp: string
  level: 'ERROR' | 'WARN' | 'INFO' | 'DEBUG'
  message: string
  source: string
  isAnomaly: boolean
  anomalyScore?: number
  rootCauses?: string[]
  recommendations?: string[]
  metadata?: Record<string, string>
  llmAnalyzed?: boolean // 是否已进行LLM分析
  analyzedAt?: string   // LLM分析时间
}

// Filter configuration
export interface LogFilter {
  timeRange: '1h' | '6h' | '24h' | '7d' | '30d' | 'all'
  source: string
  anomalyOnly: boolean
  analyzedOnly?: boolean // 新增：是否只显示已分析的日志
}

// API response types
export interface AnalysisResultsResponse {
  results: LogDisplay[]
  total: number
  page: number
  pageSize: number
}

// Realtime message types
export interface RealtimeMessage {
  type: 'new_anomaly' | 'stats_update' | 'system_alert' | 'connection_established' | 'pong'
  data: any
  timestamp: string
}

// Component props types
// Chart options type
export interface EChartsOption {
  [key: string]: any
}