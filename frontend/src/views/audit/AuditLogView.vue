<template>
  <div class="page">
    <PageHeader title="操作审计" description="所有登录、越权拒绝与关键操作的只追加记录, 任何人不可修改或删除">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="操作者 / 描述 / 资源标识" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.result" placeholder="判定结果" clearable @change="handleSearch">
          <el-option label="允许" value="allowed" />
          <el-option label="拒绝" value="denied" />
        </el-select>
        <el-select v-model="query.resource" placeholder="资源类型" clearable @change="handleSearch">
          <el-option v-for="item in resourceOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-input v-model="query.action" placeholder="动作, 如 repair.finish" clearable @keyup.enter="handleSearch" />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatDateTime(row.occurred_at) }}</template>
        </el-table-column>
        <el-table-column label="操作者" width="150">
          <template #default="{ row }">
            {{ row.actor_name || row.actor_username || '匿名' }}
            <div class="sub-text">{{ row.actor_username || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="当时角色" width="100">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ roleLabel(row.role_snapshot) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="动作" width="170" />
        <el-table-column prop="resource" label="资源" width="90" />
        <el-table-column prop="resource_id" label="资源标识" width="150" />
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag :type="row.result === 'allowed' ? 'success' : 'danger'" size="small">
              {{ row.result === 'allowed' ? '允许' : '拒绝' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="缺失授权" width="170">
          <template #default="{ row }">
            <el-tag v-if="row.required_permission" type="danger" size="small" effect="plain">
              {{ row.required_permission }}
            </el-tag>
            <span v-else class="sub-text">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
        <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="120" />
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
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

    <el-drawer v-model="detailVisible" title="审计记录详情" size="480px">
      <el-descriptions v-if="active" :column="1" border>
        <el-descriptions-item label="时间">{{ formatDateTime(active.occurred_at) }}</el-descriptions-item>
        <el-descriptions-item label="操作者">{{ active.actor_name || '-' }} ({{ active.actor_username || '匿名' }})</el-descriptions-item>
        <el-descriptions-item label="当时角色">{{ roleLabel(active.role_snapshot) }}</el-descriptions-item>
        <el-descriptions-item label="当时权限快照">
          <div class="perm-snapshot">{{ active.perm_snapshot }}</div>
        </el-descriptions-item>
        <el-descriptions-item label="动作">{{ active.action }}</el-descriptions-item>
        <el-descriptions-item label="资源">{{ active.resource }} / {{ active.resource_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="结果">
          <el-tag :type="active.result === 'allowed' ? 'success' : 'danger'" size="small">
            {{ active.result === 'allowed' ? '允许' : '拒绝' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="active.required_permission" label="缺失授权">
          <el-tag type="danger" size="small">{{ active.required_permission }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="描述">{{ active.description || '-' }}</el-descriptions-item>
        <el-descriptions-item label="原因">{{ active.reason || '-' }}</el-descriptions-item>
        <el-descriptions-item label="请求">{{ active.method }} {{ active.path }}</el-descriptions-item>
        <el-descriptions-item label="请求指纹">{{ active.request_hash }}</el-descriptions-item>
        <el-descriptions-item label="IP / UA">{{ active.ip }}<br />{{ active.user_agent }}</el-descriptions-item>
        <el-descriptions-item label="请求 ID">{{ active.request_id }}</el-descriptions-item>
      </el-descriptions>
      <el-alert
        class="immutable-tip"
        type="info"
        :closable="false"
        title="该表为只追加表: 应用无修改入口, 数据库触发器拒绝任何 UPDATE/DELETE, 权限变更也不会改写这里的历史快照。"
      />
    </el-drawer>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import { auditApi } from '@/api/auth'
import { formatDateTime } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const resourceOptions = [
  { label: '认证', value: 'auth' },
  { label: '路灯', value: 'lamp' },
  { label: '故障', value: 'fault' },
  { label: '维修', value: 'repair' },
  { label: '状态看板', value: 'status' },
  { label: '导出', value: 'export' },
  { label: '用户', value: 'user' },
  { label: '审计', value: 'audit' },
]

const roleNames = { registrar: '登记人员', repair: '维修人员', admin: '管理岗', '-': '匿名/系统' }
function roleLabel(role) {
  return roleNames[role] || role
}

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(auditApi.list, {
  keyword: '',
  result: '',
  resource: '',
  action: '',
})

const detailVisible = ref(false)
const active = ref(null)

function handleSearch() {
  search()
}
function handleReset() {
  reset()
}
function openDetail(row) {
  active.value = row
  detailVisible.value = true
}
</script>

<style scoped>
.sub-text {
  color: #909399;
  font-size: 12px;
}

.perm-snapshot {
  word-break: break-all;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  color: #606266;
}

.immutable-tip {
  margin-top: 16px;
}
</style>
