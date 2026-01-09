<template>
  <div class="logs-table">
    <!-- 操作栏 -->
    <div class="table-actions" v-if="selectable">
      <div class="selected-info">
        <span v-if="selectedLogs.length > 0">
          已选择 {{ selectedLogs.length }} 条日志
        </span>
      </div>
      <div class="action-buttons">
        <el-button 
          type="primary" 
          :disabled="selectedLogs.length === 0 || llmAnalyzing"
          :loading="llmAnalyzing"
          @click="handleLLMAnalyze"
          size="small"
        >
          <el-icon><MagicStick /></el-icon>
          LLM深度分析
        </el-button>
        <el-button 
          v-if="selectedLogs.length > 0"
          @click="clearSelection"
          size="small"
        >
          清除选择
        </el-button>
      </div>
    </div>

    <el-table 
      ref="tableRef"
      :data="logs" 
      :loading="loading"
      @row-click="handleRowClick"
      @selection-change="handleSelectionChange"
      style="width: 100%"
      row-class-name="log-row"
    >
      <!-- 选择列 -->
      <el-table-column 
        v-if="selectable"
        type="selection" 
        width="55"
      />
      
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
      
      <el-table-column prop="anomalyScore" label="异常分数" width="100">
        <template #default="{ row }">
          <span v-if="row.anomalyScore !== undefined">
            {{ row.anomalyScore.toFixed(3) }}
          </span>
          <span v-else class="text-muted">-</span>
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

      <!-- 分析状态列 -->
      <el-table-column label="分析状态" width="120" v-if="showAnalysisStatus">
        <template #default="{ row }">
          <div v-if="row.llmAnalyzed" class="analysis-status">
            <el-tag type="success" size="small">
              已分析
            </el-tag>
            <div v-if="row.analyzedAt" class="analyzed-time">
              {{ formatAnalyzedTime(row.analyzedAt) }}
            </div>
          </div>
          <el-tag v-else type="info" size="small">
            未分析
          </el-tag>
        </template>
      </el-table-column>

      <!-- LLM分析操作列 -->
      <el-table-column label="LLM分析" width="120" v-if="selectable">
        <template #default="{ row }">
          <div v-if="row.llmAnalyzed" class="llm-status">
            <el-tag type="success" size="small">
              已分析
            </el-tag>
            <div v-if="row.analyzedAt" class="analyzed-time">
              {{ formatAnalyzedTime(row.analyzedAt) }}
            </div>
          </div>
          <span v-else class="text-muted">未分析</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick, Check } from '@element-plus/icons-vue'
import type { LogDisplay } from '@/types'
import { api } from '@/services/api'

interface Props {
  logs: LogDisplay[]
  loading?: boolean
  selectable?: boolean
  showAnalysisStatus?: boolean
}

interface Emits {
  (e: 'row-click', log: LogDisplay): void
  (e: 'llm-analysis-complete', results: any[]): void
}

const props = withDefaults(defineProps<Props>(), {
  selectable: false,
  showAnalysisStatus: true
})
const emit = defineEmits<Emits>()

const tableRef = ref()
const selectedLogs = ref<LogDisplay[]>([])
const llmAnalyzing = ref(false)

function handleRowClick(row: LogDisplay) {
  emit('row-click', row)
}

function handleSelectionChange(selection: LogDisplay[]) {
  selectedLogs.value = selection
}

function clearSelection() {
  tableRef.value?.clearSelection()
  selectedLogs.value = []
}

async function handleLLMAnalyze() {
  if (selectedLogs.value.length === 0) {
    ElMessage.warning('请先选择要分析的日志')
    return
  }

  if (selectedLogs.value.length > 10) {
    ElMessage.warning('一次最多只能分析10条日志')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要对选中的 ${selectedLogs.value.length} 条日志进行LLM深度分析吗？这可能需要2-3分钟时间。`,
      'LLM深度分析',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    )

    llmAnalyzing.value = true
    
    const logIds = selectedLogs.value.map(log => log.id)
    
    // 创建进度提示消息
    let progressMessage = ElMessage({
      message: '正在初始化LLM分析...',
      type: 'info',
      duration: 0,
      showClose: false
    })
    
    try {
      // 使用流式分析
      await api.analyzeLLMStream(
        logIds,
        // 进度回调
        (event) => {
          const { type, data } = event
          
          // 更新进度消息
          progressMessage.close()
          
          let message = ''
          let messageType: 'info' | 'success' | 'warning' = 'info'
          
          switch (type) {
            case 'initializing':
              message = '正在初始化LLM分析...'
              break
            case 'analyzing':
              message = '正在调用大模型API进行深度分析...'
              break
            case 'saving':
              message = '分析完成，正在保存结果...'
              messageType = 'success'
              break
            case 'warning':
              message = data.message || '分析过程中出现警告'
              messageType = 'warning'
              break
            default:
              message = data.message || '正在处理...'
          }
          
          progressMessage = ElMessage({
            message,
            type: messageType,
            duration: 0,
            showClose: false
          })
        },
        // 完成回调
        (result) => {
          progressMessage.close()
          
          ElMessage.success(`LLM深度分析完成！已更新 ${selectedLogs.value.length} 条日志的分析结果`)
          
          // 标记已分析的日志并更新分析时间
          const now = new Date().toISOString()
          selectedLogs.value.forEach(log => {
            log.llmAnalyzed = true
            log.analyzedAt = now
          })
          
          // 通知父组件分析完成
          emit('llm-analysis-complete', result.results)
          
          // 清除选择
          clearSelection()
        },
        // 错误回调
        (error) => {
          progressMessage.close()
          throw error
        }
      )
      
    } catch (error: any) {
      // 确保关闭进度提示
      progressMessage.close()
      throw error
    }
    
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('LLM analysis error:', error)
      
      // 检查是否是超时错误
      if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
        ElMessage.error('LLM分析超时，请稍后重试。如果问题持续存在，请联系管理员。')
      } else {
        ElMessage.error(error.message || 'LLM分析失败，请稍后重试')
      }
    }
  } finally {
    llmAnalyzing.value = false
  }
}

function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
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

function formatAnalyzedTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / (1000 * 60))
    
    if (diffMins < 1) {
      return '刚刚'
    } else if (diffMins < 60) {
      return `${diffMins}分钟前`
    } else if (diffMins < 1440) {
      const hours = Math.floor(diffMins / 60)
      return `${hours}小时前`
    } else {
      return date.toLocaleDateString('zh-CN')
    }
  } catch {
    return timestamp
  }
}

function getLevelType(level: string): string {
  const typeMap: Record<string, string> = {
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
</script>

<style scoped>
.logs-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.table-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background-color: #fafafa;
  border-bottom: 1px solid #ebeef5;
}

.selected-info {
  color: #606266;
  font-size: 14px;
}

.action-buttons {
  display: flex;
  gap: 8px;
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

.text-muted {
  color: #909399;
}

.analysis-status {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
}

.llm-status {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
}

.analyzed-time {
  font-size: 11px;
  color: #909399;
  line-height: 1;
}

:deep(.el-table__header) {
  background-color: #fafafa;
}

:deep(.el-table th) {
  background-color: #fafafa !important;
}
</style>