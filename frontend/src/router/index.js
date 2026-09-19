import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'

const routes = [
  {
    path: '/',
    component: AppLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/dashboard/DashboardView.vue'),
        meta: { title: '运行看板', icon: 'DataLine' },
      },
      {
        path: 'lamps',
        name: 'lamps',
        component: () => import('@/views/lamp/LampListView.vue'),
        meta: { title: '路灯台账', icon: 'Postcard' },
      },
      {
        path: 'faults',
        name: 'faults',
        component: () => import('@/views/fault/FaultListView.vue'),
        meta: { title: '故障登记', icon: 'Warning' },
      },
      {
        path: 'repairs',
        name: 'repairs',
        component: () => import('@/views/repair/RepairListView.vue'),
        meta: { title: '维修记录录入', icon: 'Tools' },
      },
      {
        path: 'status',
        name: 'status',
        component: () => import('@/views/status/StatusLampView.vue'),
        meta: { title: '维修状态查询', icon: 'Search' },
      },
      {
        path: 'status/track',
        name: 'status-track',
        component: () => import('@/views/status/StatusTrackView.vue'),
        meta: { title: '维修进度追踪', icon: 'Guide' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.afterEach((to) => {
  document.title = to.meta?.title ? `${to.meta.title} - 路灯故障登记系统` : '路灯故障登记系统'
})

export default router
