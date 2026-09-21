<template>
  <div class="page">
    <PageHeader title="审计日志" description="全部写操作与越权拒绝的只读留痕; 记录只增不改, 权限变更也无法隐藏或改写历史">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.actor_username" placeholder="操作人" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.result" placeholder="结果" clearable @change="handleSearch" style="width: 130px">
          <el-option label="已放行" value="allowed" />
          <el-option label="被拒绝" value="denied" />
        </el-select>
        <el-input v-model="query.action" placeholder="权限点/动作 如 repair:finish" clearable @keyup.enter="handleSearch" />
        <el-input v-model="query.keyword" placeholder="单号 / 路径 / 明细" clearable @keyup.enter="handleSearch" />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          @change="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作人" width="150">
          <template #default="{ row }">
            {{ row.actor_username || '匿名' }}
            <div class="text-muted" style="font-size: 12px">{{ roleLabel(row.actor_role) }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="动作/权限点" width="170">
          <template #default="{ row }">
            <span class="mono">{{ row.action }}</span>
          </template>
        </el-table-column>
        <el-table-column label="资源" min-width="200">
          <template #default="{ row }">
            <span>{{ row.resource_type }}{{ row.resource_id ? '#' + row.resource_id : '' }}</span>
            <span v-if="row.resource_no" class="text-muted"> ({{ row.resource_no }})</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="110">
          <template #default="{ row }">
            <el-tag :type="row.result === 'denied' ? 'danger' : 'success'" size="small">
              {{ row.result === 'denied' ? '被拒绝' : '已放行' }}
            </el-tag>
            <span class="text-muted" style="margin-left: 6px">{{ row.status_code }}</span>
          </template>
        </el-table-column>
        <el-table-column label="缺失授权 / 明细" min-width="260">
          <template #default="{ row }">
            <div v-if="row.result === 'denied'" class="denied-perm">
              <el-tag type="danger" size="small" effect="dark" class="mono">{{ row.missing_permission || '未认证' }}</el-tag>
            </div>
            <div v-else-if="row.detail" class="text-muted">{{ row.detail }}</div>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="请求" width="220">
          <template #default="{ row }">
            <span class="mono" style="font-size: 12px">{{ row.method }} {{ row.path }}</span>
            <div class="text-muted" style="font-size: 12px">{{ row.client_ip }} · {{ row.request_id?.slice(0, 8) }}</div>
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
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import { authApi } from '@/api/auth'
import { useListPage } from '@/composables/useListPage'
import { formatDateTime } from '@/utils/format'

const dateRange = ref([])

const { loading, rows, total, query, load, changePage, changePageSize } = useListPage(authApi.listAuditLogs, {
  actor_username: '',
  result: '',
  action: '',
  keyword: '',
  start_date: '',
  end_date: '',
})

function roleLabel(role) {
  return { registrar: '登记人员', repairman: '维修人员', manager: '管理岗' }[role] || role || ''
}

function applyDateRange() {
  query.start_date = dateRange.value?.[0] ?? ''
  query.end_date = dateRange.value?.[1] ?? ''
}

function handleSearch() {
  applyDateRange()
  query.page = 1
  load()
}

function handleReset() {
  dateRange.value = []
  query.actor_username = ''
  query.result = ''
  query.action = ''
  query.keyword = ''
  query.start_date = ''
  query.end_date = ''
  handleSearch()
}
</script>

<style scoped>
.mono {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
}
.denied-perv {
  display: flex;
  align-items: center;
}
</style>
