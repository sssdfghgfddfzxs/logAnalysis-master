<template>
  <div class="logs-list">
    <HeaderNav />
    
    <div class="logs-content">
      <div class="page-header">
        <h1>日志列表</h1>
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
          
          <el-select 
            v-model="filters.analyzedOnly" 
            @change="updateFilters"
            style="width: 120px;"
            placeholder="分析状态"
          >
            <el-option label="全部日志" :value="undefined" />
            <el-option label="已分析" :value="true" />
            <el-option label="未分析" :value="false" />
          </el-select>
          
          <el-checkbox v-model="filters.anomalyOnly" @change="updateFilters">
            仅显示异常
          </el-checkbox>
        </div>
      </div>
      
      <LogsTable 
        :logs="logs" 
        :loading="loading"
        @row-click="showLogDetail" 
      />
      
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next, jumper"
          @current-change="handlePageChange"
        />
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
import { ref, reactive, onMounted, onUnmounted } from 'vue'
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
const refreshInterval = ref<number | null>(null)

// 定时刷新间隔（秒）
const REFRESH_INTERVAL = 30

const filters = reactive<LogFilter>({
  timeRange: '24h',
  source: 'all',
  anomalyOnly: false,
  analyzedOnly: undefined
})

async function fetchLogs() {
  loading.value = true
  try {
    // 构建包含分页信息的筛选条件
    const filterWithPagination = {
      ...filters,
      page: currentPage.value,
      limit: pageSize.value
    }
    
    // 使用新的getLogs API获取所有日志（包括分析状态）
    const response = await apiService.getLogs(filterWithPagination)
    logs.value = response.results
    total.value = response.total
  } catch (error) {
    console.error('Failed to fetch logs:', error)
    ElMessage.error('获取日志列表失败')
  } finally {
    loading.value = false
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

function startAutoRefresh() {
  // 清除现有定时器
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
  
  // 设置新的定时器
  refreshInterval.value = window.setInterval(() => {
    fetchLogs()
  }, REFRESH_INTERVAL * 1000)
}

function stopAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

onMounted(async () => {
  await fetchLogs()
  
  // 启动定时刷新
  startAutoRefresh()
  
  console.log(`LogsList initialized with ${REFRESH_INTERVAL}s auto-refresh`)
})

onUnmounted(() => {
  // 清理定时器
  stopAutoRefresh()
})
</script>

<style scoped>
.logs-list {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.logs-content {
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
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: center;
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