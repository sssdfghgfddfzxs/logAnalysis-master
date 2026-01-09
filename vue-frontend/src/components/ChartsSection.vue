<template>
  <div class="charts-section">
    <div class="charts-grid">
      <div class="chart-item">
        <h3>异常趋势</h3>
        <div class="chart-placeholder">
          <p>图表组件将在后续任务中实现</p>
        </div>
      </div>
      
      <div class="chart-item">
        <h3>服务分布</h3>
        <div class="chart-placeholder">
          <p>图表组件将在后续任务中实现</p>
        </div>
      </div>
      
      <div class="chart-item">
        <h3>异常级别分布</h3>
        <div class="chart-placeholder">
          <p>图表组件将在后续任务中实现</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DashboardChartData } from '@/types'

interface Props {
  chartData: DashboardChartData
  loading?: boolean
}

defineProps<Props>()
</script>

<style scoped>
.charts-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.chart-item {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 16px;
}

.chart-item h3 {
  margin: 0 0 12px 0;
  font-size: 16px;
  color: #303133;
}

.chart-placeholder {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  border-radius: 4px;
  border: 2px dashed #dcdfe6;
}

.chart-placeholder p {
  color: #909399;
  font-size: 14px;
}
</style>