<template>
  <div class="page">
    <PageHeader title="用户与权限" description="维护登录账号及其岗位角色; 角色决定可操作范围, 调整后立即生效">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增用户</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="keyword" placeholder="用户名 / 姓名" clearable @keyup.enter="load" />
        <el-select v-model="roleFilter" placeholder="角色" clearable @change="load">
          <el-option v-for="item in roles" :key="item.key" :label="item.label" :value="item.key" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="load">查询</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="display_name" label="姓名" min-width="140" />
        <el-table-column label="角色" width="140">
          <template #default="{ row }">
            <el-tag :type="roleTagType(row.role)" effect="plain">{{ row.role_label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.active ? 'success' : 'info'" size="small">
              {{ row.active ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑角色</el-button>
            <el-button link :type="row.active ? 'warning' : 'success'" @click="toggleActive(row)">
              {{ row.active ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" @click="openReset(row)">重置密码</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="editing ? '编辑用户' : '新增用户'" width="460px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" :disabled="!!editing" placeholder="至少 3 个字符" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="form.display_name" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role" style="width: 100%">
            <el-option v-for="item in roles" :key="item.key" :label="`${item.label} - ${item.description}`" :value="item.key" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="!editing" label="初始密码">
          <el-input v-model="form.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resetVisible" title="重置密码" width="420px">
      <el-input v-model="resetPassword" type="password" show-password placeholder="输入新密码(至少 6 位)" />
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" @click="submitReset">确认重置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { userApi, authApi } from '@/api/auth'

const roles = ref([
  { key: 'registrar', label: '登记人员', description: '登记与查询' },
  { key: 'repair', label: '维修人员', description: '推进本人负责的记录' },
  { key: 'admin', label: '管理岗', description: '改派/例外/导出/审计' },
])

const loading = ref(false)
const rows = ref([])
const keyword = ref('')
const roleFilter = ref('')

const formVisible = ref(false)
const editing = ref(null)
const form = reactive({ username: '', display_name: '', role: 'registrar', password: '' })

const resetVisible = ref(false)
const resetTarget = ref(null)
const resetPassword = ref('')

function roleTagType(role) {
  return role === 'admin' ? 'success' : role === 'repair' ? 'warning' : 'primary'
}

async function load() {
  loading.value = true
  try {
    const data = await userApi.list({ keyword: keyword.value, role: roleFilter.value })
    rows.value = data.items || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { username: '', display_name: '', role: 'registrar', password: '' })
  formVisible.value = true
}

function openEdit(row) {
  editing.value = row
  Object.assign(form, {
    username: row.username,
    display_name: row.display_name,
    role: row.role,
    password: '',
  })
  formVisible.value = true
}

async function submitForm() {
  try {
    if (editing.value) {
      await userApi.update(editing.value.id, {
        display_name: form.display_name,
        role: form.role,
      })
      ElMessage.success('已保存, 该用户权限立即生效')
    } else {
      await userApi.create({ ...form })
      ElMessage.success('用户已创建')
    }
    formVisible.value = false
    load()
  } catch {
    /* 提示由拦截器处理 */
  }
}

async function toggleActive(row) {
  const next = !row.active
  try {
    await ElMessageBox.confirm(`确认${next ? '启用' : '停用'}用户 ${row.username}?`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await userApi.update(row.id, { active: next })
    ElMessage.success(next ? '已启用' : '已停用, 该账号立即无法继续操作')
    load()
  } catch {
    /* ignore */
  }
}

function openReset(row) {
  resetTarget.value = row
  resetPassword.value = ''
  resetVisible.value = true
}

async function submitReset() {
  try {
    await userApi.resetPassword(resetTarget.value.id, { password: resetPassword.value })
    ElMessage.success('密码已重置')
    resetVisible.value = false
  } catch {
    /* ignore */
  }
}

onMounted(async () => {
  try {
    const catalog = await authApi.catalog()
    if (Array.isArray(catalog.roles) && catalog.roles.length) {
      roles.value.splice(0, roles.value.length, ...catalog.roles)
    }
  } catch {
    /* 使用内置角色兜底 */
  }
  load()
})
</script>
