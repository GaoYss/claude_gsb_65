<template>
  <el-container class="app-layout">
    <el-aside width="220px" class="app-aside">
      <div class="app-brand">
        <el-icon :size="20"><Sunny /></el-icon>
        <span>路灯故障登记系统</span>
      </div>
      <el-menu :default-active="activeMenu" router class="app-menu">
        <el-menu-item v-for="item in visibleMenus" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="app-header">
        <div class="app-header__title">{{ currentTitle }}</div>
        <div class="app-header__extra">
          <el-tag type="success" effect="plain">{{ today }}</el-tag>
          <el-dropdown @command="handleCommand">
            <span class="app-user">
              <el-icon><UserFilled /></el-icon>
              <span class="app-user__name">{{ auth.displayName }}</span>
              <el-tag size="small" type="warning" effect="dark">{{ auth.roleLabel }}</el-tag>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>{{ auth.profile?.username }}</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

// 菜单与路由 meta.permission 同源, 按权限点过滤, 后端无权限的页面入口不出现。
const menus = [
  { path: '/dashboard', title: '运行看板', icon: 'DataLine', permission: 'status:read' },
  { path: '/lamps', title: '路灯台账', icon: 'Postcard', permission: 'lamp:read' },
  { path: '/faults', title: '故障登记', icon: 'Warning', permission: 'fault:read' },
  { path: '/repairs', title: '维修记录', icon: 'Tools', permission: 'repair:read' },
  { path: '/status', title: '维修状态查询', icon: 'Search', permission: 'status:read' },
  { path: '/status/track', title: '维修进度追踪', icon: 'Guide', permission: 'status:read' },
  { path: '/admin/users', title: '用户与授权', icon: 'UserFilled', permission: 'user:manage' },
  { path: '/admin/audit', title: '审计日志', icon: 'Document', permission: 'audit:read' },
]

const visibleMenus = computed(() => menus.filter((item) => auth.can(item.permission)))

const activeMenu = computed(() => {
  const matched = visibleMenus.value
    .filter((item) => route.path === item.path || route.path.startsWith(`${item.path}/`))
    .sort((a, b) => b.path.length - a.path.length)
  return matched[0]?.path ?? route.path
})

const currentTitle = computed(() => route.meta?.title ?? '路灯故障登记系统')
const today = computed(() => formatDate(new Date()))

async function handleCommand(command) {
  if (command !== 'logout') return
  try {
    await ElMessageBox.confirm('确认退出当前账号?', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }
  await auth.logout()
  router.replace('/login')
}
</script>

<style scoped>
.app-layout {
  height: 100vh;
}

.app-aside {
  background-color: #1f2d3d;
  display: flex;
  flex-direction: column;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 60px;
  padding: 0 16px;
  color: #fff;
  font-weight: 600;
  font-size: 15px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.app-menu {
  flex: 1;
  border-right: none;
  background-color: transparent;
}

.app-menu :deep(.el-menu-item) {
  color: #c0c4cc;
}

.app-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background-color: #409eff;
}

.app-menu :deep(.el-menu-item:hover) {
  background-color: rgba(64, 158, 255, 0.2);
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  border-bottom: 1px solid var(--app-border);
}

.app-header__title {
  font-size: 16px;
  font-weight: 600;
}

.app-header__extra {
  display: flex;
  align-items: center;
  gap: 14px;
}

.app-user {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #303133;
  outline: none;
}

.app-user__name {
  font-size: 14px;
  font-weight: 600;
}

.app-main {
  background-color: var(--app-bg);
  padding: 16px;
}
</style>
