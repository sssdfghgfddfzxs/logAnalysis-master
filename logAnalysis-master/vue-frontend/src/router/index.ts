import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '@/views/Dashboard.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Dashboard,
      meta: {
        title: '仪表板'
      }
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import('@/views/LogsList.vue'),
      meta: {
        title: '日志列表'
      }
    },
    {
      path: '/analysis',
      name: 'analysis',
      component: () => import('@/views/AnalysisResults.vue'),
      meta: {
        title: '分析结果'
      }
    },
    {
      path: '/alerts',
      name: 'alerts',
      component: () => import('@/views/AlertRules.vue'),
      meta: {
        title: '告警规则'
      }
    }
  ]
})

// Navigation guard to set page title
router.beforeEach((to, _from, next) => {
  if (to.meta?.title) {
    document.title = `${to.meta.title} - 智能日志异常分析系统`
  }
  next()
})

export default router