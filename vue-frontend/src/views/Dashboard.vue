<template>
  <div class="dashboard">
    <HeaderNav />
    
    <div class="dashboard-content">
      <!-- 连接状态指示器 -->
      <div class="status-bar">
        <div class="status-item">
          <span class="status-dot" :class="{ 'active': !loading }"></span>
          <span>{{ loading ? '数据更新中...' : '数据已同步' }}</span>
        </div>
        <div class="status-item">
          <span class="refresh-time">最后更新: {{ lastUpdateTime }}</span>
        </div>
      </div>

      <!-- 核心统计数据 -->
      <div class="stats-section">
        <h2 class="section-title">系统概览</h2>
        <div class="stats-grid">
          <div class="stat-card primary">
            <div class="stat-icon">📊</div>
            <div class="stat-content">
              <div class="stat-value">{{ dashboardStore.stats.total_logs }}</div>
              <div class="stat-label">总日志数</div>
            </div>
          </div>
          
          <div class="stat-card danger">
            <div class="stat-icon">⚠️</div>
            <div class="stat-content">
              <div class="stat-value">{{ dashboardStore.stats.anomaly_count }}</div>
              <div class="stat-label">异常数量</div>
            </div>
          </div>
          
          <div class="stat-card" :class="dashboardStore.stats.anomaly_rate > 0.1 ? 'danger' : 'success'">
            <div class="stat-icon">📈</div>
            <div class="stat-content">
              <div class="stat-value">{{ (dashboardStore.stats.anomaly_rate * 100).toFixed(2) }}%</div>
              <div class="stat-label">异常率</div>
            </div>
          </div>
          
          <div class="stat-card info">
            <div class="stat-icon">🤖</div>
            <div class="stat-content">
              <div class="stat-value">LLM</div>
              <div class="stat-label">AI分析引擎</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Charts Section -->
      <div class="charts-section">
        <h2 class="section-title">趋势分析</h2>
        <ChartsSection :chart-data="dashboardStore.chartData" :loading="dashboardStore.loading" />
      </div>

      <!-- 日志列表 -->
      <div class="logs-section">
        <div class="section-header">
          <h2 class="section-title">最近日志</h2>
          <div class="filters">
            
            <el-checkbox v-model="filters.anomalyOnly" @change="updateFilters" size="small">
              仅显示异常
            </el-checkbox>
            
            <el-button type="primary" @click="$router.push('/logs')">
              查看全部
            </el-button>
          </div>
        </div>
        
        <div class="logs-table-container">
          <el-table 
            :data="logs" 
            :loading="loading"
            @row-click="showLogDetail"
            style="width: 100%"
            row-class-name="log-row"
            max-height="400"
          >
            <el-table-column prop="timestamp" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.timestamp) }}
              </template>
            </el-table-column>
            
            <el-table-column prop="level" label="级别" width="80">
              <template #default="{ row }">
                <el-tag :type="getLevelType(row.level)" size="small">
                  {{ row.level }}
                </el-tag>
              </template>
            </el-table-column>
            
            <el-table-column prop="source" label="来源" width="120" />
            
            <el-table-column prop="message" label="消息" min-width="300">
              <template #default="{ row }">
                <div class="message-cell">
                  {{ truncateMessage(row.message) }}
                </div>
              </template>
            </el-table-column>
            
            <el-table-column prop="isAnomaly" label="状态" width="80">
              <template #default="{ row }">
                <el-tag v-if="row.isAnomaly" type="danger" size="small">
                  异常
                </el-tag>
                <el-tag v-else type="success" size="small">
                  正常
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
        
        <div class="pagination">
          <el-pagination
            v-model:current-page="currentPage"
            :page-size="pageSize"
            :total="total"
            layout="total, prev, pager, next"
            @current-change="handlePageChange"
            small
          />
        </div>
      </div>

      <!-- 分析结果摘要 -->
      <div v-if="anomalyLogs.length > 0" class="analysis-section">
        <h2 class="section-title">异常分析</h2>
        <div class="analysis-summary">
          <div class="summary-card">
            <h3>🔍 检测到 {{ anomalyLogs.length }} 个异常</h3>
            <div v-if="topRootCauses.length > 0" class="causes">
              <h4>主要问题:</h4>
              <ul>
                <li v-for="cause in topRootCauses.slice(0, 3)" :key="cause">
                  {{ truncateText(cause, 100) }}
                </li>
              </ul>
            </div>
            <div v-if="topRecommendations.length > 0" class="recommendations">
              <h4>建议措施:</h4>
              <ul>
                <li v-for="rec in topRecommendations.slice(0, 3)" :key="rec">
                  {{ truncateText(rec, 100) }}
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Log Detail Modal -->
    <LogDetailModal 
      v-if="selectedLog" 
      :log="selectedLog" 
      @close="closeDetail" 
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useDashboardStore } from '@/stores/dashboard'
import { apiService } from '@/services/api'
import type { LogDisplay, LogFilter } from '@/types'

import HeaderNav from '@/components/HeaderNav.vue'
import ChartsSection from '@/components/ChartsSection.vue'
import LogDetailModal from '@/components/LogDetailModal.vue'

const dashboardStore = useDashboardStore()
const selectedLog = ref<LogDisplay | null>(null)
const logs = ref<LogDisplay[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const lastUpdateTime = ref('')
const refreshInterval = ref<number | null>(null)

// 定时刷新间隔（秒）
const REFRESH_INTERVAL = 30

const filters = reactive({
  timeRange: '24h' as const,
  source: 'all',
  anomalyOnly: false,
  level: 'all' // Add level filter separately from LogFilter type
})

const anomalyLogs = computed(() => 
  logs.value.filter(log => log.isAnomaly)
)

const topRootCauses = computed(() => {
  const causes = anomalyLogs.value
    .flatMap(log => log.rootCauses || [])
    .reduce((acc, cause) => {
      acc[cause] = (acc[cause] || 0) + 1
      return acc
    }, {} as Record<string, number>)
  
  return Object.entries(causes)
    .sort(([,a], [,b]) => b - a)
    .slice(0, 5)
    .map(([cause]) => cause)
})

const topRecommendations = computed(() => {
  const recommendations = anomalyLogs.value
    .flatMap(log => log.recommendations || [])
    .reduce((acc, rec) => {
      acc[rec] = (acc[rec] || 0) + 1
      return acc
    }, {} as Record<string, number>)
  
  return Object.entries(recommendations)
    .sort(([,a], [,b]) => b - a)
    .slice(0, 5)
    .map(([rec]) => rec)
})

async function fetchLogs() {
  loading.value = true
  try {
    const filterWithPagination = {
      timeRange: filters.timeRange,
      source: filters.source,
      anomalyOnly: filters.anomalyOnly,
      page: currentPage.value,
      limit: pageSize.value
    }
    
    const response = await apiService.getAnalysisResults(filterWithPagination)
    
    // Apply level filter on client side if needed
    let filteredResults = response.results
    if (filters.level !== 'all') {
      filteredResults = response.results.filter(log => log.level === filters.level)
    }
    
    logs.value = filteredResults
    total.value = response.total
    
    // 更新最后刷新时间
    lastUpdateTime.value = new Date().toLocaleTimeString('zh-CN')
    
  } catch (error) {
    console.error('Failed to fetch logs:', error)
    ElMessage.error('获取日志列表失败')
  } finally {
    loading.value = false
  }
}

async function refreshData() {
  // 同时刷新统计数据和日志列表
  await Promise.all([
    dashboardStore.fetchDashboardData(),
    fetchLogs()
  ])
}

function startAutoRefresh() {
  // 清除现有定时器
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
  
  // 设置新的定时器
  refreshInterval.value = window.setInterval(() => {
    refreshData()
  }, REFRESH_INTERVAL * 1000)
}

function stopAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

function updateFilters() {
  currentPage.value = 1
  fetchLogs()
}

function handlePageChange(page: number) {
  currentPage.value = page
  fetchLogs()
}

function showLogDetail(log: LogDisplay) {
  selectedLog.value = log
}

function closeDetail() {
  selectedLog.value = null
}

function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return timestamp
  }
}

function getLevelType(level: string): 'danger' | 'warning' | 'info' | 'success' | 'primary' {
  const typeMap: Record<string, 'danger' | 'warning' | 'info' | 'success' | 'primary'> = {
    ERROR: 'danger',
    WARN: 'warning',
    INFO: 'info',
    DEBUG: 'info'
  }
  return typeMap[level] || 'info'
}

function truncateMessage(message: string, maxLength: number = 100): string {
  if (!message || typeof message !== 'string') return ''
  if (message.length <= maxLength) return message
  return message.substring(0, maxLength) + '...'
}

function truncateText(text: string, maxLength: number): string {
  if (!text || typeof text !== 'string') return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

onMounted(async () => {
  // 初始数据加载
  await refreshData()
  
  // 启动定时刷新
  startAutoRefresh()
  
  console.log(`Dashboard initialized with ${REFRESH_INTERVAL}s auto-refresh using pure LLM analysis`)
})

onUnmounted(() => {
  // 清理定时器
  stopAutoRefresh()
})
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.dashboard-content {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: white;
  border-radius: 8px;
  padding: 12px 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  border-left: 4px solid #409eff;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #e4e7ed;
  transition: background-color 0.3s;
}

.status-dot.active {
  background-color: #67c23a;
}

.refresh-time {
  font-size: 13px;
  color: #909399;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}

.stats-section,
.charts-section,
.logs-section,
.analysis-section {
  margin-bottom: 24px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 16px;
  border-left: 4px solid #409eff;
}

.stat-card.primary {
  border-left-color: #409eff;
}

.stat-card.danger {
  border-left-color: #f56c6c;
}

.stat-card.success {
  border-left-color: #67c23a;
}

.stat-card.info {
  border-left-color: #909399;
}

.stat-icon {
  font-size: 24px;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #606266;
  margin-top: 4px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filters {
  display: flex;
  gap: 16px;
  align-items: center;
}

.logs-table-container {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  margin-bottom: 16px;
}

:deep(.log-row) {
  cursor: pointer;
}

:deep(.log-row:hover) {
  background-color: #f5f7fa;
}

.message-cell {
  word-break: break-word;
  line-height: 1.4;
}

.pagination {
  display: flex;
  justify-content: center;
}

.analysis-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.summary-card {
  background: #fef0f0;
  border: 1px solid #fbc4c4;
  border-radius: 6px;
  padding: 16px;
}

.summary-card h3 {
  margin: 0 0 12px 0;
  color: #f56c6c;
  font-size: 16px;
}

.summary-card h4 {
  margin: 12px 0 8px 0;
  color: #303133;
  font-size: 14px;
}

.causes ul,
.recommendations ul {
  margin: 0;
  padding-left: 20px;
}

.causes li,
.recommendations li {
  margin-bottom: 4px;
  color: #606266;
  font-size: 13px;
  line-height: 1.4;
}

@media (max-width: 768px) {
  .dashboard-content {
    padding: 16px;
  }
  
  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .filters {
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}
</style>