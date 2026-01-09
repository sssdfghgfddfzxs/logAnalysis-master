<template>
  <div class="analysis-results">
    <HeaderNav />
    
    <div class="analysis-content">
      <div class="page-header">
        <h1>分析结果</h1>
        <div class="filters">
          <el-select 
            v-model="filters.timeRange" 
            @change="updateFilters"
            style="width: 140px;"
            placeholder="选择时间范围"
          >
            <el-option label="最近1小时" value="1h" />
            <el-option label="最近6小时" value="6h" />
            <el-option label="最近24小时" value="24h" />
            <el-option label="最近7天" value="7d" />
            <el-option label="最近30天" value="30d" />
            <el-option label="全部" value="all" />
          </el-select>
          
          <el-checkbox v-model="filters.anomalyOnly" @change="updateFilters">
            仅显示异常
          </el-checkbox>
        </div>
      </div>
      
      <div class="results-grid">
        <div class="analysis-logs">
          <h2>分析结果 ({{ total }})</h2>
          <div v-if="logs.length === 0" class="no-data">
            <el-empty description="暂无分析结果" />
          </div>
          <LogsTable 
            v-else
            :logs="logs" 
            :loading="loading"
            :selectable="false"
            :show-analysis-status="false"
            @row-click="showLogDetail"
          />
          
          <!-- 分页 -->
          <div class="pagination" v-if="total > 0">
            <el-pagination
              v-model:current-page="currentPage"
              :page-size="pageSize"
              :total="total"
              layout="total, prev, pager, next, jumper"
              @current-change="handlePageChange"
            />
          </div>
        </div>
        
        <div class="analysis-summary">
          <h2>分析摘要</h2>
          <div class="summary-cards">
            <div class="summary-card">
              <h3>异常检测</h3>
              <p>总日志数: {{ logs.length }}</p>
              <p>检测到 {{ anomalyCount }} 个异常日志</p>
              <p>异常率: {{ anomalyRate }}%</p>
            </div>
            
            <div class="summary-card" v-if="topRootCauses.length > 0">
              <h3>主要问题</h3>
              <ul>
                <li v-for="cause in topRootCauses" :key="cause">
                  {{ cause }}
                </li>
              </ul>
            </div>
            
            <div class="summary-card" v-if="topRecommendations.length > 0">
              <h3>建议措施</h3>
              <ul>
                <li v-for="rec in topRecommendations" :key="rec">
                  {{ rec }}
                </li>
              </ul>
            </div>
            
            <div class="summary-card" v-if="topRootCauses.length === 0 && topRecommendations.length === 0">
              <h3>系统状态</h3>
              <p>✅ 系统运行正常</p>
              <p>✅ 未发现异常模式</p>
              <p>✅ 所有日志均为正常状态</p>
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
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { apiService } from '@/services/api'
import type { LogDisplay, LogFilter } from '@/types'

import HeaderNav from '@/components/HeaderNav.vue'
import LogsTable from '@/components/LogsTable.vue'
import LogDetailModal from '@/components/LogDetailModal.vue'

const logs = ref<LogDisplay[]>([])
const loading = ref(false)
const selectedLog = ref<LogDisplay | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filters = reactive<LogFilter>({
  timeRange: '24h',
  source: 'all',
  anomalyOnly: false
})

const anomalyLogs = computed(() => 
  logs.value.filter(log => log.isAnomaly)
)

const anomalyCount = computed(() => anomalyLogs.value.length)

const anomalyRate = computed(() => {
  if (logs.value.length === 0) return 0
  return ((anomalyCount.value / logs.value.length) * 100).toFixed(2)
})

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

async function fetchAnalysisResults() {
  loading.value = true
  try {
    // 获取分析结果，支持分页和筛选
    const filterWithPagination = {
      ...filters,
      page: currentPage.value,
      limit: pageSize.value
    }
    
    const response = await apiService.getAnalysisResults(filterWithPagination)
    logs.value = response.results
    total.value = response.total
  } catch (error) {
    console.error('Failed to fetch analysis results:', error)
    ElMessage.error('获取分析结果失败')
  } finally {
    loading.value = false
  }
}

function updateFilters() {
  currentPage.value = 1
  fetchAnalysisResults()
}

function handlePageChange(page: number) {
  currentPage.value = page
  fetchAnalysisResults()
}

function showLogDetail(log: LogDisplay) {
  selectedLog.value = log
}

function handleLLMAnalysisComplete(results: any[]) {
  console.log('LLM analysis completed:', results)
  // 刷新数据以显示最新的分析结果
  fetchAnalysisResults()
}

function closeDetail() {
  selectedLog.value = null
}

let refreshInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await fetchAnalysisResults()
  
  // Set up polling for updates every 30 seconds
  refreshInterval = setInterval(fetchAnalysisResults, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.analysis-results {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.analysis-content {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.filters {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.filters .el-select {
  min-width: 120px;
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}

.results-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 24px;
}

.analysis-logs,
.analysis-summary {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.analysis-logs h2,
.analysis-summary h2 {
  margin: 0 0 16px 0;
  font-size: 18px;
  color: #303133;
}

.summary-cards {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary-card {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border-left: 4px solid #409eff;
}

.summary-card h3 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.summary-card p {
  margin: 4px 0;
  font-size: 13px;
  color: #606266;
}

.summary-card ul {
  margin: 8px 0 0 0;
  padding-left: 16px;
}

.summary-card li {
  font-size: 13px;
  color: #606266;
  margin-bottom: 4px;
}

.no-data {
  padding: 40px;
  text-align: center;
  color: #909399;
}

@media (max-width: 1024px) {
  .results-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .filters {
    flex-wrap: wrap;
    gap: 12px;
    width: 100%;
  }
  
  .filters .el-select {
    min-width: 100px;
    flex: 1;
  }
}
</style>