<template>
  <el-dialog
    v-model="visible"
    title="日志详情"
    width="800px"
    @close="handleClose"
  >
    <div class="log-detail">
      <div class="detail-section">
        <h3>基本信息</h3>
        <div class="detail-grid">
          <div class="detail-item">
            <label>时间:</label>
            <span>{{ formatTime(log.timestamp) }}</span>
          </div>
          <div class="detail-item">
            <label>级别:</label>
            <el-tag :type="getLevelType(log.level)">{{ log.level }}</el-tag>
          </div>
          <div class="detail-item">
            <label>来源:</label>
            <span>{{ log.source }}</span>
          </div>
          <div class="detail-item">
            <label>状态:</label>
            <el-tag v-if="log.isAnomaly" type="danger">异常</el-tag>
            <el-tag v-else type="success">正常</el-tag>
          </div>
          <div v-if="log.anomalyScore !== undefined" class="detail-item">
            <label>异常分数:</label>
            <span>{{ log.anomalyScore.toFixed(3) }}</span>
          </div>
        </div>
      </div>

      <div class="detail-section">
        <h3>消息内容</h3>
        <div class="message-content">
          {{ log.message }}
        </div>
      </div>

      <div v-if="log.metadata && Object.keys(log.metadata).length > 0" class="detail-section">
        <h3>元数据</h3>
        <div class="metadata-grid">
          <div v-for="(value, key) in log.metadata" :key="key" class="metadata-item">
            <label>{{ key }}:</label>
            <span>{{ value }}</span>
          </div>
        </div>
      </div>

      <div v-if="log.isAnomaly && log.rootCauses && log.rootCauses.length > 0" class="detail-section">
        <h3>根因分析</h3>
        <ul class="causes-list">
          <li v-for="cause in log.rootCauses" :key="cause">
            {{ cause }}
          </li>
        </ul>
      </div>

      <div v-if="log.isAnomaly && log.recommendations && log.recommendations.length > 0" class="detail-section">
        <h3>建议措施</h3>
        <ul class="recommendations-list">
          <li v-for="rec in log.recommendations" :key="rec">
            {{ rec }}
          </li>
        </ul>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
      <el-button type="primary" @click="copyToClipboard">复制详情</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { LogDisplay } from '@/types'

interface Props {
  log: LogDisplay
}

interface Emits {
  (e: 'close'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = ref(true)

watch(() => props.log, () => {
  visible.value = true
})

function handleClose() {
  visible.value = false
  emit('close')
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

function getLevelType(level: string): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  const typeMap: Record<string, 'primary' | 'success' | 'warning' | 'info' | 'danger'> = {
    ERROR: 'danger',
    WARN: 'warning',
    INFO: 'info',
    DEBUG: 'info'
  }
  return typeMap[level] || 'info'
}

function copyToClipboard() {
  const details = [
    `时间: ${formatTime(props.log.timestamp)}`,
    `级别: ${props.log.level}`,
    `来源: ${props.log.source}`,
    `状态: ${props.log.isAnomaly ? '异常' : '正常'}`,
    props.log.anomalyScore !== undefined ? `异常分数: ${props.log.anomalyScore.toFixed(3)}` : '',
    `消息: ${props.log.message}`,
    props.log.rootCauses?.length ? `根因: ${props.log.rootCauses.join(', ')}` : '',
    props.log.recommendations?.length ? `建议: ${props.log.recommendations.join(', ')}` : ''
  ].filter(Boolean).join('\n')

  navigator.clipboard.writeText(details).then(() => {
    ElMessage.success('详情已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}
</script>

<style scoped>
.log-detail {
  max-height: 600px;
  overflow-y: auto;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h3 {
  margin: 0 0 12px 0;
  font-size: 16px;
  color: #303133;
  border-bottom: 1px solid #e4e7ed;
  padding-bottom: 8px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-item label {
  font-weight: 500;
  color: #606266;
  min-width: 60px;
}

.message-content {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 6px;
  border-left: 4px solid #409eff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
}

.metadata-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 8px;
}

.metadata-item {
  display: flex;
  gap: 8px;
  padding: 8px;
  background: #f8f9fa;
  border-radius: 4px;
}

.metadata-item label {
  font-weight: 500;
  color: #606266;
  min-width: 80px;
}

.causes-list,
.recommendations-list {
  margin: 0;
  padding-left: 20px;
}

.causes-list li,
.recommendations-list li {
  margin-bottom: 8px;
  color: #606266;
  line-height: 1.5;
}

.causes-list {
  border-left: 4px solid #f56c6c;
  padding-left: 16px;
  background: #fef0f0;
  padding: 12px 16px;
  border-radius: 4px;
}

.recommendations-list {
  border-left: 4px solid #67c23a;
  padding-left: 16px;
  background: #f0f9ff;
  padding: 12px 16px;
  border-radius: 4px;
}
</style>