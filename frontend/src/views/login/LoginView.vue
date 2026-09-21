<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <el-icon :size="28"><Sunny /></el-icon>
        <div>
          <div class="login-title">路灯故障登记系统</div>
          <div class="login-subtitle">请使用分配的账号登录</div>
        </div>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="handleLogin">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" size="large" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            :prefix-icon="Lock"
            size="large"
            show-password
            autocomplete="current-password"
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-button type="primary" size="large" class="login-btn" :loading="loading" @click="handleLogin">
          登 录
        </el-button>
      </el-form>

      <div class="login-hint">
        <div class="login-hint__title">演示账号(初始口令 streetlight123)</div>
        <div class="login-hint__row"><el-tag size="small">登记人员</el-tag> registrar01 · 王建国</div>
        <div class="login-hint__row"><el-tag size="small" type="warning">维修人员</el-tag> repair01 · 刘志强 / repair02 · 陈鹏</div>
        <div class="login-hint__row"><el-tag size="small" type="success">管理岗</el-tag> manager01 · 调度主管</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const formRef = ref()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await auth.login(form.username.trim(), form.password)
    ElMessage.success(`欢迎, ${auth.displayName}(${auth.roleLabel})`)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.replace(redirect)
  } catch (error) {
    // 401 提示由登录接口错误信息直接展示
    ElMessage.error(error.message || '登录失败')
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
  background: linear-gradient(135deg, #1f2d3d 0%, #2c3e50 100%);
  padding: 16px;
}

.login-card {
  width: 400px;
  max-width: 100%;
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  color: #1f2d3d;
}

.login-title {
  font-size: 18px;
  font-weight: 700;
}

.login-subtitle {
  font-size: 13px;
  color: #909399;
  margin-top: 2px;
}

.login-btn {
  width: 100%;
}

.login-hint {
  margin-top: 22px;
  padding-top: 16px;
  border-top: 1px dashed var(--app-border, #dcdfe6);
  font-size: 12px;
  color: #606266;
  line-height: 2;
}

.login-hint__title {
  font-weight: 600;
  margin-bottom: 4px;
}

.login-hint__row {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
