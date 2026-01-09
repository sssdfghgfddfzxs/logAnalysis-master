import axios from 'axios'
import type { DashboardStats, LogFilter, AnalysisResultsResponse, ChartData } from '@/types'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000, // 默认30秒超时
})

// 创建一个专门用于LLM分析的axios实例，使用更长的超时时间
const llmApi = axios.create({
  baseURL: '/api/v1',
  timeout: 150000, // 150秒超时，给LLM分析足够的时间
})

// Request interceptor for both instances
const setupInterceptors = (instance: typeof api) => {
  instance.interceptors.request.use(
    (config) => {
      // Add any auth headers here if needed
      return config
    },
    (error) => {
      return Promise.reject(error)
    }
  )

  instance.interceptors.response.use(
    (response) => {
      return response.data
    },
    (error) => {
      console.error('API Error:', error)
      
      if (error.response?.status === 401) {
        // Handle unauthorized access
        console.error('Unauthorized access')
      } else if (error.response?.status >= 500) {
        // Handle server errors
        console.error('Server error')
      }
      
      return Promise.reject(error)
    }
  )
}

// Setup interceptors for both instances
setupInterceptors(api)
setupInterceptors(llmApi)

export const apiService = {
  // Dashboard APIs
  async getDashboardStats(period: string = '24h'): Promise<DashboardStats> {
    return api.get(`/dashboard/stats?period=${period}`)
  },

  async getAnomalyTrends(period: string = '24h'): Promise<ChartData> {
    return api.get(`/dashboard/trends?period=${period}`)
  },

  // Logs APIs - 新增
  async getLogs(filters: LogFilter & { page?: number; limit?: number }): Promise<AnalysisResultsResponse> {
    const params = new URLSearchParams()
    
    if (filters.timeRange !== 'all') {
      params.append('time_range', filters.timeRange)
    }
    if (filters.source !== 'all') {
      params.append('source', filters.source)
    }
    if (filters.anomalyOnly) {
      params.append('anomaly_only', 'true')
    }
    // 添加分析状态筛选
    if (filters.analyzedOnly !== undefined) {
      params.append('analyzed', filters.analyzedOnly.toString())
    }
    if (filters.page) {
      params.append('page', filters.page.toString())
    }
    if (filters.limit) {
      params.append('limit', filters.limit.toString())
    }

    const response = await api.get(`/logs?${params.toString()}`)
    const data = response as any
    
    // Transform the API response to LogDisplay format
    const transformedResults = data.results.map((result: any) => ({
      id: result.id,
      timestamp: result.timestamp,
      level: result.level,
      message: result.message,
      source: result.source,
      isAnomaly: result.is_anomaly || false,
      anomalyScore: result.anomaly_score,
      rootCauses: result.root_causes?.causes || [],
      recommendations: result.recommendations?.recommendations || [],
      metadata: result.metadata || {},
      llmAnalyzed: result.is_analyzed,
      analyzedAt: result.analyzed_at
    }))

    return {
      results: transformedResults,
      total: data.total,
      page: data.page,
      pageSize: data.limit
    }
  },

  // 获取已分析的日志（用于仪表板）
  async getAnalyzedLogs(limit: number = 50): Promise<any> {
    const response = await api.get(`/logs/analyzed?limit=${limit}`)
    const data = response as any
    
    // Transform the API response to LogDisplay format
    const transformedResults = data.results.map((result: any) => ({
      id: result.id,
      timestamp: result.timestamp,
      level: result.level,
      message: result.message,
      source: result.source,
      isAnomaly: result.is_anomaly || false,
      anomalyScore: result.anomaly_score,
      rootCauses: result.root_causes?.causes || [],
      recommendations: result.recommendations?.recommendations || [],
      metadata: result.metadata || {},
      llmAnalyzed: true, // 已分析的日志
      analyzedAt: result.analyzed_at
    }))

    return {
      results: transformedResults,
      total: data.total
    }
  },

  // Analysis APIs
  async getAnalysisResults(filters: LogFilter & { page?: number; limit?: number }): Promise<AnalysisResultsResponse> {
    const params = new URLSearchParams()
    
    if (filters.timeRange !== 'all') {
      params.append('period', filters.timeRange)
    }
    if (filters.source !== 'all') {
      params.append('source', filters.source)
    }
    if (filters.anomalyOnly) {
      params.append('anomaly_only', 'true')
    }
    if (filters.page) {
      params.append('page', filters.page.toString())
    }
    if (filters.limit) {
      params.append('limit', filters.limit.toString())
    }

    const response = await api.get(`/analysis/results?${params.toString()}`)
    const data = response as any // Type assertion since we know the interceptor returns response.data
    
    // Transform the nested API response to flat LogDisplay format
    const transformedResults = data.results.map((result: any) => ({
      id: result.log?.id || result.id,
      timestamp: result.log?.timestamp || result.analyzed_at,
      level: result.log?.level || 'INFO',
      message: result.log?.message || '',
      source: result.log?.source || '',
      isAnomaly: result.is_anomaly || false,
      anomalyScore: result.anomaly_score,
      rootCauses: result.root_causes?.causes || [],
      recommendations: result.recommendations?.recommendations || [],
      metadata: result.log?.metadata || {},
      llmAnalyzed: !!(result.root_causes?.causes?.some((cause: string) => cause.includes('🤖')) || 
                     result.recommendations?.recommendations?.some((rec: string) => rec.includes('🤖'))),
      analyzedAt: result.analyzed_at // 添加分析时间
    }))

    return {
      results: transformedResults,
      total: data.total,
      page: data.page,
      pageSize: data.limit
    }
  },

  // Log submission API
  async submitLog(logData: any): Promise<void> {
    return api.post('/logs', logData)
  },

  // AI Analysis APIs - 统一使用LLM分析
  async analyzeLogs(logIds: string[]): Promise<any> {
    // 现在统一使用LLM分析，移除useLLM参数
    return api.post('/ai/analyze', { log_ids: logIds })
  },

  // 流式LLM分析 - 使用Server-Sent Events
  async analyzeLLMStream(
    logIds: string[], 
    onProgress?: (event: { type: string; data: any }) => void,
    onComplete?: (result: any) => void,
    onError?: (error: any) => void
  ): Promise<void> {
    const url = `${api.defaults.baseURL}/ai/analyze` // 修正URL
    
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'text/event-stream',
        },
        body: JSON.stringify({ 
          log_ids: logIds, 
          stream: true 
        })
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()

      if (!reader) {
        throw new Error('Response body is not readable')
      }

      let buffer = ''
      
      while (true) {
        const { done, value } = await reader.read()
        
        if (done) break
        
        buffer += decoder.decode(value, { stream: true })
        
        // 处理SSE事件
        const lines = buffer.split('\n')
        buffer = lines.pop() || '' // 保留不完整的行
        
        for (const line of lines) {
          let currentEventType = 'progress' // 默认事件类型
          
          if (line.startsWith('event: ')) {
            currentEventType = line.substring(7).trim()
            continue
          }
          
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.substring(6))
              const eventType = data.stage || currentEventType
              
              // 调用进度回调
              if (onProgress) {
                onProgress({ type: eventType, data })
              }
              
              // 处理完成事件
              if (eventType === 'completed' || eventType === 'done') {
                if (onComplete && data.results) {
                  onComplete(data)
                }
              }
              
              // 处理错误事件
              if (eventType === 'failed' || eventType === 'error') {
                if (onError) {
                  onError(new Error(data.message || 'LLM analysis failed'))
                }
                return
              }
              
            } catch (e) {
              console.warn('Failed to parse SSE data:', line)
            }
          }
        }
      }
      
    } catch (error) {
      console.error('Stream LLM analysis error:', error)
      if (onError) {
        onError(error)
      }
      throw error
    }
  },

  // Health check
  async healthCheck(): Promise<{ status: string; time: string }> {
    return api.get('/health')
  },

  // Generic HTTP methods for alert rules and other APIs
  async get(url: string, config?: any): Promise<any> {
    return api.get(url, config)
  },

  async post(url: string, data?: any, config?: any): Promise<any> {
    return api.post(url, data, config)
  },

  async put(url: string, data?: any, config?: any): Promise<any> {
    return api.put(url, data, config)
  },

  async delete(url: string, config?: any): Promise<any> {
    return api.delete(url, config)
  },

  async patch(url: string, data?: any, config?: any): Promise<any> {
    return api.patch(url, data, config)
  }
}

export { apiService as api }


export default api