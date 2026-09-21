<template>
  <div class="page">
    <PageHeader title="用户与授权" description="维护登录账号、角色与启停状态; 角色变更会立即让该账号重新登录, 并记入审计">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增用户</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="用户名 / 姓名 / 班组" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.role" placeholder="角色" clearable @change="handleSearch">
          <el-option v-for="role in roles" :key="role.value" :label="role.label" :value="role.value" />
        </el-select>
        <el-select v-model="activeFilter" placeholder="状态" clearable @change="handleActiveFilter">
          <el-option label="启用" :value="true" />
          <el-option label="停用" :value="false" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="display_name" label="姓名" width="120" />
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="roleTagType(row.role)" effect="dark">{{ row.role_label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="team" label="班组" min-width="140" show-overflow-tooltip />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.active ? 'success' : 'info'">{{ row.active ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_login_at" label="最近登录" width="180">
          <template #default="{ row }">{{ formatDateTime(row.last_login_at) || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑/改角色</el-button>
            <el-button link type="warning" @click="openResetPassword(row)">重置密码</el-button>
            <el-button :link="true" :type="row.active ? 'danger' : 'success'" @click="toggleActive(row)">
              {{ row.active ? '停用' : '启用' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <DataPagination
        :page="query.page"
        :page-size="query.page_size"
        :total="total"
        @page-change="changePage"
        @size-change="changePageSize"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑用户' : '新增用户'" width="480px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="!!editing" placeholder="登录用户名" />
        </el-form-item>
        <el-form-item label="姓名" prop="display_name">
          <el-input v-model="form.display_name" placeholder="真实姓名(维修人员用于归属匹配)" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" style="width: 100%">
            <el-option v-for="role in roles" :key="role.value" :label="role.label" :value="role.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="班组">
          <el-input v-model="form.team" placeholder="选填, 如 市政照明一班" />
        </el-form-item>
        <el-form-item v-if="!editing" label="初始密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="初始登录密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import { authApi } from '@/api/auth'
import { useListPage } from '@/composables/useListPage'
import { formatDateTime } from '@/utils/format'

const roles = [
  { value: 'registrar', label: '登记人员' },
  { value: 'repairman', label: '维修人员' },
  { value: 'manager', label: '管理岗' },
]

function roleTagType(role) {
  return role === 'manager' ? 'success' : role === 'repairman' ? 'warning' : 'primary'
}

const fetcher = (params) => authApi.listUsers(params)
const { loading, rows, total, query, load, changePage, changePageSize } = useListPage(fetcher, {
  keyword: '',
  role: '',
  active: undefined,
})

const activeFilter = ref(null)

const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref(null)
const formRef = ref()
const form = reactive({ username: '', display_name: '', role: 'registrar', team: '', password: '' })

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  display_name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  password: [{ required: true, message: '请输入初始密码', trigger: 'blur' }],
}

function handleSearch() {
  query.page = 1
  load()
}
function handleReset() {
  query.keyword = ''
  query.role = ''
  activeFilter.value = null
  query.active = undefined
  handleSearch()
}
function handleActiveFilter() {
  query.active = activeFilter.value
  handleSearch()
}

function openCreate() {
  editing.value = null
  Object.assign(form, { username: '', display_name: '', role: 'registrar', team: '', password: '' })
  dialogVisible.value = true
}

function openEdit(row) {
  editing.value = { ...row }
  Object.assign(form, {
    username: row.username,
    display_name: row.display_name,
    role: row.role,
    team: row.team || '',
    password: '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    if (editing.value) {
      const payload = { display_name: form.display_name, role: form.role, team: form.team }
      await authApi.updateUser(editing.value.id, payload)
      ElMessage.success('已保存, 若角色发生变更该账号需重新登录')
    } else {
      await authApi.createUser({ ...form })
      ElMessage.success('用户已创建')
    }
    dialogVisible.value = false
    load()
  } catch (error) {
    // 统一提示
  } finally {
    saving.value = false
  }
}

async function openResetPassword(row) {
  try {
    const { value } = await ElMessageBox.prompt(`为 ${row.display_name} 设置新密码`, '重置密码', {
      confirmButtonText: '确认重置',
      cancelButtonText: '取消',
      inputType: 'password',
      inputPlaceholder: '新密码',
    })
    if (!value) {
      ElMessage.warning('密码不能为空')
      return
    }
    await authApi.updateUser(row.id, { password: value })
    ElMessage.success('密码已重置, 该账号需重新登录')
  } catch (error) {
    // 取消
  }
}

async function toggleActive(row) {
  const next = !row.active
  try {
    await ElMessageBox.confirm(
      `确认${next ? '启用' : '停用'}账号 ${row.username}?${next ? '' : ' 停用后该账号会被立即强制下线。'}`,
      '状态变更',
      { type: 'warning', confirmButtonText: '确认', cancelButtonText: '取消' },
    )
  } catch (error) {
    return
  }
  try {
    await authApi.updateUser(row.id, { active: next })
    ElMessage.success(`账号已${next ? '启用' : '停用'}`)
    load()
  } catch (error) {
    // 统一提示
  }
}

onMounted(load)
</script>
