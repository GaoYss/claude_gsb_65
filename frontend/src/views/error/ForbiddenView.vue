<template>
  <div class="forbidden-page">
    <el-icon :size="56" color="#e6a23c"><WarningFilled /></el-icon>
    <h2>无权访问该页面</h2>
    <p class="subtitle">你的岗位角色没有被授予该功能的操作范围, 如需开通请联系管理岗。</p>
    <div v-if="missing" class="missing-box">
      <span class="missing-label">缺少授权点：</span>
      <el-tag type="danger" size="large">{{ missing }}</el-tag>
    </div>
    <el-button type="primary" @click="$router.replace('/dashboard')">返回看板</el-button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const missing = computed(() => route.query.permission || '')
</script>

<style scoped>
.forbidden-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 70vh;
  text-align: center;
}

.subtitle {
  color: #909399;
  margin: 0;
}

.missing-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 8px 0;
}

.missing-label {
  color: #606266;
  font-size: 14px;
}
</style>
