<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <el-icon :size="28"><Sunny /></el-icon>
        <h1>路灯故障登记系统</h1>
      </div>
      <p class="login-sub">请使用分配给你的岗位账号登录</p>

      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @keyup.enter="handleLogin">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="用户名" size="large" :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            size="large"
            show-password
            :prefix-icon="Lock"
          />
        </el-form-item>
        <el-button type="primary" size="large" class="login-btn" :loading="loading" @click="handleLogin">
          登 录
        </el-button>
      </el-form>

      <el-divider>演示账号</el-divider>
      <div class="login-accounts">
        <el-tag
          v-for="item in accounts"
          :key="item.username"
          :type="item.type"
          effect="plain"
          class="account-tag"
          @click="fill(item)"
        >
          {{ item.label }}：{{ item.username }} / {{ item.password }}
        </el-tag>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, Sunny, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const formRef = ref()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const accounts = [
  { label: '登记人员', username: 'registrar', password: 'registrar123', type: 'primary' },
  { label: '维修人员', username: 'repairman', password: 'repair123', type: 'warning' },
  { label: '管理岗', username: 'admin', password: 'admin123', type: 'success' },
]

function fill(item) {
  form.username = item.username
  form.password = item.password
}

async function handleLogin() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    const redirect = route.value.query.redirect
    router.replace(redirect ? decodeURIComponent(redirect) : '/dashboard')
  } catch (error) {
    // 错误提示由拦截器统一处理
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #2c3e50 60%, #409eff 140%);
  padding: 16px;
}

.login-card {
  width: 400px;
  max-width: 100%;
  background: #fff;
  border-radius: 12px;
  padding: 32px 32px 24px;
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.24);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #1f2d3d;
}

.login-brand h1 {
  font-size: 20px;
  margin: 0;
}

.login-sub {
  color: #909399;
  font-size: 13px;
  margin: 8px 0 20px;
}

.login-btn {
  width: 100%;
}

.login-accounts {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.account-tag {
  cursor: pointer;
  justify-content: flex-start;
  height: auto;
  padding: 6px 10px;
  white-space: normal;
}
</style>
