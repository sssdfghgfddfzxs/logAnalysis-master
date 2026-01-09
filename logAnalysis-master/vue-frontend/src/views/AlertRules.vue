<template>
  <div class="alert-rules">
    <HeaderNav />
    
    <div class="alert-rules-container">
      <div class="header">
        <h1>告警规则管理</h1>
        <el-button type="primary" @click="showCreateDialog = true">
          <el-icon><Plus /></el-icon>
          创建告警规则
        </el-button>
      </div>

    <!-- 告警规则列表 -->
    <el-card class="rules-card">
      <template #header>
        <div class="card-header">
          <span>告警规则列表</span>
          <el-button text @click="loadAlertRules">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-table :data="alertRules" v-loading="loading" stripe>
        <el-table-column prop="name" label="规则名称" min-width="150" />
        <el-table-column label="条件" min-width="200">
          <template #default="{ row }">
            <el-tag v-if="row.condition.anomaly_score_threshold" type="warning" size="small">
              异常评分 > {{ row.condition.anomaly_score_threshold }}
            </el-tag>
            <el-tag v-if="row.condition.levels" type="info" size="small" class="ml-1">
              级别: {{ row.condition.levels.join(', ') }}
            </el-tag>
            <el-tag v-if="row.condition.sources" type="success" size="small" class="ml-1">
              来源: {{ row.condition.sources.join(', ') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="通知渠道" min-width="120">
          <template #default="{ row }">
            <el-tag 
              v-for="channel in row.notification_channels" 
              :key="channel" 
              :type="getChannelType(channel) as any"
              size="small"
              class="mr-1"
            >
              {{ getChannelName(channel) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-switch 
              v-model="row.is_active" 
              @change="toggleRuleStatus(row)"
              :loading="row.updating"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="testRule(row)" :loading="row.testing">
              测试
            </el-button>
            <el-button size="small" type="primary" @click="editRule(row)">
              编辑
            </el-button>
            <el-button size="small" type="danger" @click="deleteRule(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑告警规则对话框 -->
    <el-dialog 
      v-model="showCreateDialog" 
      :title="editingRule ? '编辑告警规则' : '创建告警规则'"
      width="600px"
      @close="resetForm"
    >
      <el-form :model="ruleForm" :rules="rules" ref="ruleFormRef" label-width="120px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input 
            v-model="ruleForm.description" 
            type="textarea" 
            :rows="2"
            placeholder="请输入规则描述（可选）" 
          />
        </el-form-item>

        <el-divider content-position="left">触发条件</el-divider>
        
        <el-form-item label="异常评分阈值" prop="anomaly_score_threshold">
          <el-slider 
            v-model="ruleForm.condition.anomaly_score_threshold" 
            :min="0" 
            :max="1" 
            :step="0.1"
            show-input
            :format-tooltip="(val: number) => `${val} (${val >= 0.8 ? '高' : val >= 0.5 ? '中' : '低'})`"
          />
          <div class="form-help">当异常评分超过此阈值时触发告警</div>
        </el-form-item>

        <el-form-item label="日志级别">
          <el-checkbox-group v-model="ruleForm.condition.levels">
            <el-checkbox label="ERROR">错误</el-checkbox>
            <el-checkbox label="WARN">警告</el-checkbox>
            <el-checkbox label="INFO">信息</el-checkbox>
            <el-checkbox label="DEBUG">调试</el-checkbox>
          </el-checkbox-group>
          <div class="form-help">选择要监控的日志级别（不选择表示监控所有级别）</div>
        </el-form-item>

        <el-form-item label="日志来源">
          <el-input 
            v-model="sourceInput" 
            placeholder="输入日志来源，按回车添加"
            @keyup.enter="addSource"
          />
          <div class="sources-container mt-2">
            <el-tag 
              v-for="source in ruleForm.condition.sources" 
              :key="source"
              closable
              @close="removeSource(source)"
              class="mr-1 mb-1"
            >
              {{ source }}
            </el-tag>
          </div>
          <div class="form-help">指定要监控的日志来源（不指定表示监控所有来源）</div>
        </el-form-item>

        <el-form-item label="时间窗口">
          <el-select v-model="ruleForm.condition.time_window_minutes" placeholder="选择时间窗口">
            <el-option label="5分钟" :value="5" />
            <el-option label="10分钟" :value="10" />
            <el-option label="15分钟" :value="15" />
            <el-option label="30分钟" :value="30" />
            <el-option label="60分钟" :value="60" />
          </el-select>
          <div class="form-help">在此时间窗口内满足条件的异常数量</div>
        </el-form-item>

        <el-form-item label="最小异常数量">
          <el-input-number 
            v-model="ruleForm.condition.min_anomaly_count" 
            :min="1" 
            :max="100"
            placeholder="最小异常数量"
          />
          <div class="form-help">在时间窗口内至少检测到多少个异常才触发告警</div>
        </el-form-item>

        <el-divider content-position="left">通知设置</el-divider>

        <el-form-item label="通知渠道" prop="notification_channels">
          <el-checkbox-group v-model="ruleForm.notification_channels">
            <el-checkbox 
              v-for="channel in availableChannels" 
              :key="channel.name"
              :label="channel.name"
            >
              {{ channel.icon }} {{ channel.display_name }}
            </el-checkbox>
          </el-checkbox-group>
          <div class="form-help">选择告警通知的发送渠道</div>
        </el-form-item>

        <el-form-item label="启用规则">
          <el-switch v-model="ruleForm.is_active" />
          <div class="form-help">是否立即启用此告警规则</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">
          {{ editingRule ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { api } from '@/services/api'
import HeaderNav from '@/components/HeaderNav.vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const showCreateDialog = ref(false)
const editingRule = ref<any>(null)
const alertRules = ref<any[]>([])
const availableChannels = ref<any[]>([])
const sourceInput = ref('')

// 表单数据
const ruleForm = reactive({
  name: '',
  description: '',
  condition: {
    anomaly_score_threshold: 0.8,
    levels: [] as string[],
    sources: [] as string[],
    time_window_minutes: 15,
    min_anomaly_count: 1
  },
  notification_channels: [] as string[],
  is_active: true
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入规则名称', trigger: 'blur' },
    { min: 2, max: 100, message: '规则名称长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  notification_channels: [
    { required: true, message: '请选择至少一个通知渠道', trigger: 'change' }
  ]
}

const ruleFormRef = ref()

// 生命周期
onMounted(() => {
  loadAlertRules()
  loadNotificationChannels()
})

// 方法
const loadAlertRules = async () => {
  loading.value = true
  try {
    const response = await api.get('/alert-rules')
    // 由于axios拦截器已经返回了response.data，所以这里直接访问rules
    alertRules.value = response.rules || []
  } catch (error) {
    console.error('Failed to load alert rules:', error)
    ElMessage.error('加载告警规则失败')
  } finally {
    loading.value = false
  }
}

const loadNotificationChannels = async () => {
  try {
    const response = await api.get('/notification-channels')
    // 由于axios拦截器已经返回了response.data，所以这里直接访问channels
    availableChannels.value = response.channels || []
  } catch (error) {
    console.error('Failed to load notification channels:', error)
  }
}

const saveRule = async () => {
  if (!ruleFormRef.value) return
  
  try {
    await ruleFormRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const payload = {
      name: ruleForm.name,
      description: ruleForm.description,
      condition: ruleForm.condition,
      notification_channels: ruleForm.notification_channels,
      is_active: ruleForm.is_active
    }

    if (editingRule.value) {
      await api.put(`/alert-rules/${editingRule.value.id}`, payload)
      ElMessage.success('告警规则更新成功')
    } else {
      await api.post('/alert-rules', payload)
      ElMessage.success('告警规则创建成功')
    }

    showCreateDialog.value = false
    loadAlertRules()
  } catch (error) {
    console.error('Failed to save alert rule:', error)
    ElMessage.error(editingRule.value ? '更新告警规则失败' : '创建告警规则失败')
  } finally {
    saving.value = false
  }
}

const editRule = (rule: any) => {
  editingRule.value = rule
  ruleForm.name = rule.name
  ruleForm.description = rule.description || ''
  ruleForm.condition = { ...rule.condition }
  ruleForm.notification_channels = [...rule.notification_channels]
  ruleForm.is_active = rule.is_active
  showCreateDialog.value = true
}

const deleteRule = async (rule: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除告警规则 "${rule.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await api.delete(`/alert-rules/${rule.id}`)
    ElMessage.success('告警规则删除成功')
    loadAlertRules()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete alert rule:', error)
      ElMessage.error('删除告警规则失败')
    }
  }
}

const testRule = async (rule: any) => {
  rule.testing = true
  try {
    const response = await api.post(`/alert-rules/${rule.id}/test`)
    const results = response.results
    
    let message = '测试告警发送结果：\n'
    for (const [channel, result] of Object.entries(results)) {
      message += `${getChannelName(channel)}: ${result}\n`
    }
    
    ElMessage.success(message)
  } catch (error) {
    console.error('Failed to test alert rule:', error)
    ElMessage.error('测试告警规则失败')
  } finally {
    rule.testing = false
  }
}

const toggleRuleStatus = async (rule: any) => {
  rule.updating = true
  try {
    await api.put(`/alert-rules/${rule.id}`, {
      name: rule.name,
      condition: rule.condition,
      notification_channels: rule.notification_channels,
      is_active: rule.is_active
    })
    ElMessage.success(`告警规则已${rule.is_active ? '启用' : '禁用'}`)
  } catch (error) {
    console.error('Failed to toggle rule status:', error)
    ElMessage.error('更新告警规则状态失败')
    // 回滚状态
    rule.is_active = !rule.is_active
  } finally {
    rule.updating = false
  }
}

const addSource = () => {
  if (sourceInput.value.trim() && !ruleForm.condition.sources.includes(sourceInput.value.trim())) {
    ruleForm.condition.sources.push(sourceInput.value.trim())
    sourceInput.value = ''
  }
}

const removeSource = (source: string) => {
  const index = ruleForm.condition.sources.indexOf(source)
  if (index > -1) {
    ruleForm.condition.sources.splice(index, 1)
  }
}

const resetForm = () => {
  editingRule.value = null
  ruleForm.name = ''
  ruleForm.description = ''
  ruleForm.condition = {
    anomaly_score_threshold: 0.8,
    levels: [],
    sources: [],
    time_window_minutes: 15,
    min_anomaly_count: 1
  }
  ruleForm.notification_channels = []
  ruleForm.is_active = true
  sourceInput.value = ''
}

const getChannelName = (channel: string) => {
  const channelMap: Record<string, string> = {
    email: '邮件',
    dingtalk: '钉钉',
    webhook: 'Webhook'
  }
  return channelMap[channel] || channel
}

const getChannelType = (channel: string) => {
  const typeMap: Record<string, string> = {
    email: 'primary',
    dingtalk: 'success',
    webhook: 'info'
  }
  return typeMap[channel] || 'info'
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN')
}
</script>

<style scoped>
.alert-rules {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.alert-rules-container {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h1 {
  margin: 0;
  color: #303133;
}

.rules-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ml-1 {
  margin-left: 4px;
}

.mr-1 {
  margin-right: 4px;
}

.mb-1 {
  margin-bottom: 4px;
}

.mt-2 {
  margin-top: 8px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.sources-container {
  min-height: 32px;
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  padding: 8px;
  background-color: #fafafa;
}

.sources-container:empty::before {
  content: '暂无来源过滤';
  color: #c0c4cc;
  font-size: 12px;
}
</style>