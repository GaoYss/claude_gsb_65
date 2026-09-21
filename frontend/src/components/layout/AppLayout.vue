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
          <el-tag effect="plain" :type="roleTagType">{{ auth.user?.role_label || '-' }}</el-tag>
          <el-dropdown @command="handleCommand">
            <span class="user-trigger">
              <el-icon><UserFilled /></el-icon>
              {{ auth.user?.display_name || auth.user?.username }}
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
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

    <el-dialog v-model="pwdVisible" title="修改密码" width="420px">
      <el-form :model="pwdForm" label-width="90px">
        <el-form-item label="原密码">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdLoading" @click="submitPassword">确认</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

// 菜单与路由 meta 同源, 仅展示当前角色有权进入的功能。
const menus = [
  { path: '/dashboard', title: '运行看板', icon: 'DataLine', permission: 'status:view' },
  { path: '/lamps', title: '路灯台账', icon: 'Postcard', permission: 'lamp:view' },
  { path: '/faults', title: '故障登记', icon: 'Warning', permission: 'fault:view' },
  { path: '/repairs', title: '维修记录录入', icon: 'Tools', permission: 'repair:view' },
  { path: '/status', title: '维修状态查询', icon: 'Search', permission: 'status:view' },
  { path: '/status/track', title: '维修进度追踪', icon: 'Guide', permission: 'status:view' },
  { path: '/audit', title: '操作审计', icon: 'DocumentChecked', permission: 'audit:view' },
  { path: '/users', title: '用户与权限', icon: 'UserFilled', permission: 'user:manage' },
]

const visibleMenus = computed(() => menus.filter((item) => auth.can(item.permission)))

const activeMenu = computed(() => {
  const matched = visibleMenus.value
    .filter((item) => route.path === item.path || route.path.startsWith(`${item.path}/`))
    .sort((a, b) => b.path.length - a.path.length)
  return matched[0]?.path ?? route.path
})

const currentTitle = computed(() => route.meta?.title ?? '路灯故障登记系统')

const roleTagType = computed(() => {
  switch (auth.user?.role) {
    case 'admin':
      return 'success'
    case 'repair':
      return 'warning'
    default:
      return 'primary'
  }
})

const pwdVisible = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '' })

async function handleCommand(command) {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确认退出登录?', '提示', { type: 'warning' })
    } catch {
      return
    }
    await auth.logout()
    ElMessage.success('已退出登录')
    router.replace('/login')
  } else if (command === 'password') {
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdVisible.value = true
  }
}

async function submitPassword() {
  if (!pwdForm.old_password || !pwdForm.new_password) {
    ElMessage.warning('请填写原密码与新密码')
    return
  }
  pwdLoading.value = true
  try {
    await authApi.changePassword(pwdForm)
    ElMessage.success('密码已修改, 请重新登录')
    pwdVisible.value = false
    await auth.logout()
    router.replace('/login')
  } catch (error) {
    // 提示由拦截器处理
  } finally {
    pwdLoading.value = false
  }
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
  gap: 12px;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #303133;
  outline: none;
}

.app-main {
  background-color: var(--app-bg);
  padding: 16px;
}
</style>
