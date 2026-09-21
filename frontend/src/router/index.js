import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

// meta.permission: 进入该路由所需的权限点, 与后端接口权限同源。
const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/LoginView.vue'),
    meta: { title: '登录', public: true },
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
        meta: { title: '运行看板', icon: 'DataLine', permission: 'status:read' },
      },
      {
        path: 'lamps',
        name: 'lamps',
        component: () => import('@/views/lamp/LampListView.vue'),
        meta: { title: '路灯台账', icon: 'Postcard', permission: 'lamp:read' },
      },
      {
        path: 'faults',
        name: 'faults',
        component: () => import('@/views/fault/FaultListView.vue'),
        meta: { title: '故障登记', icon: 'Warning', permission: 'fault:read' },
      },
      {
        path: 'repairs',
        name: 'repairs',
        component: () => import('@/views/repair/RepairListView.vue'),
        meta: { title: '维修记录', icon: 'Tools', permission: 'repair:read' },
      },
      {
        path: 'status',
        name: 'status',
        component: () => import('@/views/status/StatusLampView.vue'),
        meta: { title: '维修状态查询', icon: 'Search', permission: 'status:read' },
      },
      {
        path: 'status/track',
        name: 'status-track',
        component: () => import('@/views/status/StatusTrackView.vue'),
        meta: { title: '维修进度追踪', icon: 'Guide', permission: 'status:read' },
      },
      {
        path: 'admin/users',
        name: 'admin-users',
        component: () => import('@/views/admin/UserListView.vue'),
        meta: { title: '用户与授权', icon: 'UserFilled', permission: 'user:manage' },
      },
      {
        path: 'admin/audit',
        name: 'admin-audit',
        component: () => import('@/views/admin/AuditLogView.vue'),
        meta: { title: '审计日志', icon: 'Document', permission: 'audit:read' },
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

// 登录守卫: 未登录跳登录页; 已登录但权限点未加载则先拉 /auth/me; 无权限拦截并提示。
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (to.meta.public) {
    if (to.name === 'login' && auth.isLoggedIn) return { path: '/' }
    return true
  }

  if (!auth.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (!auth.loaded) {
    try {
      await auth.fetchMe()
    } catch (error) {
      auth.reset()
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }

  const permission = to.meta.permission
  if (permission && !auth.can(permission)) {
    ElMessage.error(`无权访问该页面: 缺少授权 ${permission}`)
    return { path: '/dashboard' }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta?.title ? `${to.meta.title} - 路灯故障登记系统` : '路灯故障登记系统'
})

export default router
