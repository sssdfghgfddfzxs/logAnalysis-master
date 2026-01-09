<template>
  <header class="header-nav">
    <div class="nav-container">
      <div class="nav-brand">
        <h1>智能日志异常分析系统</h1>
      </div>
      
      <nav class="nav-menu">
        <router-link to="/" class="nav-item" active-class="active">
          <el-icon><Monitor /></el-icon>
          仪表板
        </router-link>
        <router-link to="/logs" class="nav-item" active-class="active">
          <el-icon><Document /></el-icon>
          日志列表
        </router-link>
        <router-link to="/analysis" class="nav-item" active-class="active">
          <el-icon><DataAnalysis /></el-icon>
          分析结果
        </router-link>
        <router-link to="/alerts" class="nav-item" active-class="active">
          <el-icon><Bell /></el-icon>
          告警规则
        </router-link>
      </nav>
      
      <div class="nav-actions">
        <!-- Connection Status Indicator -->
        <div class="connection-status" :class="{ 'connected': isConnected, 'disconnected': !isConnected }">
          <el-tooltip :content="connectionTooltip" placement="bottom">
            <el-icon :size="16">
              <Connection v-if="isConnected" />
              <Close v-else />
            </el-icon>
          </el-tooltip>
        </div>
        
        <el-badge :value="anomalyCount" :hidden="anomalyCount === 0" type="danger">
          <el-button type="text" @click="showNotifications">
            <el-icon><Bell /></el-icon>
          </el-button>
        </el-badge>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useDashboardStore } from '@/stores/dashboard'
import { Monitor, Document, DataAnalysis, Bell, Connection, Close } from '@element-plus/icons-vue'

const dashboardStore = useDashboardStore()
const isConnected = ref(true) // 由于移除了WebSocket，默认为true

const anomalyCount = computed(() => dashboardStore.stats.anomaly_count)

const connectionTooltip = computed(() => {
  return '定时轮询模式 (30秒刷新)'
})

function showNotifications() {
  ElMessage.info('通知功能开发中...')
}

onMounted(() => {
  // 纯LLM架构使用定时轮询，不需要WebSocket连接状态
  isConnected.value = true
})
</script>

<style scoped>
.header-nav {
  background: white;
  border-bottom: 1px solid #e4e7ed;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.nav-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 60px;
}

.nav-brand h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.nav-menu {
  display: flex;
  gap: 32px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  text-decoration: none;
  color: #606266;
  border-radius: 6px;
  transition: all 0.2s;
  font-size: 14px;
}

.nav-item:hover {
  color: #409eff;
  background-color: #f0f9ff;
}

.nav-item.active {
  color: #409eff;
  background-color: #e1f3ff;
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.connection-status {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  transition: all 0.3s ease;
}

.connection-status.connected {
  background-color: #f0f9ff;
  color: #67c23a;
}

.connection-status.disconnected {
  background-color: #fef0f0;
  color: #f56c6c;
}

.connection-status:hover {
  transform: scale(1.1);
}

@media (max-width: 768px) {
  .nav-container {
    padding: 0 16px;
  }
  
  .nav-brand h1 {
    font-size: 16px;
  }
  
  .nav-menu {
    gap: 16px;
  }
  
  .nav-item {
    padding: 6px 12px;
    font-size: 13px;
  }
}
</style>