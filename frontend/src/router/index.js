import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/403',
    name: 'forbidden',
    component: () => import('@/views/error/ForbiddenView.vue'),
    meta: { title: '无权限' },
  },
  {
    path: '/',
    component: AppLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/dashboard/DashboardView.vue'),
        meta: { title: '运行看板', icon: 'DataLine', permission: 'status:view' },
      },
      {
        path: 'lamps',
        name: 'lamps',
        component: () => import('@/views/lamp/LampListView.vue'),
        meta: { title: '路灯台账', icon: 'Postcard', permission: 'lamp:view' },
      },
      {
        path: 'faults',
        name: 'faults',
        component: () => import('@/views/fault/FaultListView.vue'),
        meta: { title: '故障登记', icon: 'Warning', permission: 'fault:view' },
      },
      {
        path: 'repairs',
        name: 'repairs',
        component: () => import('@/views/repair/RepairListView.vue'),
        meta: { title: '维修记录录入', icon: 'Tools', permission: 'repair:view' },
      },
      {
        path: 'status',
        name: 'status',
        component: () => import('@/views/status/StatusLampView.vue'),
        meta: { title: '维修状态查询', icon: 'Search', permission: 'status:view' },
      },
      {
        path: 'status/track',
        name: 'status-track',
        component: () => import('@/views/status/StatusTrackView.vue'),
        meta: { title: '维修进度追踪', icon: 'Guide', permission: 'status:view' },
      },
      {
        path: 'audit',
        name: 'audit',
        component: () => import('@/views/audit/AuditLogView.vue'),
        meta: { title: '操作审计', icon: 'DocumentChecked', permission: 'audit:view' },
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('@/views/system/UserListView.vue'),
        meta: { title: '用户与权限', icon: 'UserFilled', permission: 'user:manage' },
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

// 全局前置守卫: 未登录跳登录页; 已登录但缺少页面所需权限 -> 403 并带上缺失授权点。
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.bootstrap()

  if (to.meta.public) {
    if (to.name === 'login' && auth.isLoggedIn) return { path: '/dashboard' }
    return true
  }

  if (!auth.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  const permission = to.meta.permission
  if (permission && !auth.can(permission)) {
    return { path: '/403', query: { permission } }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta?.title ? `${to.meta.title} - 路灯故障登记系统` : '路灯故障登记系统'
})

export default router
