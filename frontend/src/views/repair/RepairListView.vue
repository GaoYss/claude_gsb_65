<template>
  <div class="page">
    <PageHeader title="维修记录录入" description="记录维修过程、耗材与费用, 完工后自动联动故障与路灯状态">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button v-permission="PERM.EXPORT_REPAIR" :icon="Download" @click="handleExport">导出 CSV</el-button>
      <el-button v-permission="PERM.REPAIR_CLAIM" type="primary" :icon="Plus" @click="openCreate">录入维修记录</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="维修单号 / 故障单号 / 路灯编号 / 维修人员" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="维修状态" clearable @change="handleSearch">
          <el-option v-for="(item, key) in REPAIR_STATUS" :key="key" :label="item.label" :value="key" />
        </el-select>
        <el-select v-model="query.result" placeholder="维修结果" clearable @change="handleSearch">
          <el-option v-for="(item, key) in REPAIR_RESULT" :key="key" :label="item.label" :value="key" />
        </el-select>
        <el-select v-model="query.repairman" placeholder="维修人员" clearable @change="handleSearch">
          <el-option v-for="item in dictStore.repairMeta.repairmen" :key="item" :label="item" :value="item" />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开工开始日期"
          end-placeholder="开工结束日期"
          @change="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="repair_no" label="维修单号" width="140" fixed="left" />
        <el-table-column prop="fault_no" label="故障单号" width="140" />
        <el-table-column prop="lamp_code" label="路灯编号" width="110" />
        <el-table-column prop="repairman" label="维修人员" width="100" />
        <el-table-column label="负责人" width="110">
          <template #default="{ row }">
            <el-tag v-if="isMine(row)" type="warning" size="small">本人</el-tag>
            <span v-else>{{ row.repairman }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="repair_team" label="维修班组" width="130" />
        <el-table-column label="开工时间" width="150">
          <template #default="{ row }">{{ formatDateTime(row.started_at) }}</template>
        </el-table-column>
        <el-table-column label="完工时间" width="150">
          <template #default="{ row }">{{ formatDateTime(row.finished_at) }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="120">
          <template #default="{ row }">{{ formatDuration(row.duration_minutes) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }"><StatusTag :dict="REPAIR_STATUS" :value="row.status" /></template>
        </el-table-column>
        <el-table-column label="结果" width="100">
          <template #default="{ row }">
            <StatusTag v-if="row.result" :dict="REPAIR_RESULT" :value="row.result" />
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="费用" width="100">
          <template #default="{ row }">{{ formatMoney(row.cost) }}</template>
        </el-table-column>
        <el-table-column prop="content" label="维修内容" min-width="180" show-overflow-tooltip />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">故障详情</el-button>
            <el-button v-if="row.status === 'ongoing' && canAdvance(row)" link type="success" @click="openFinish(row)">完成维修</el-button>
            <el-button v-if="row.status === 'ongoing' && canAdvance(row)" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="PERM.REPAIR_REASSIGN" link type="warning" @click="openReassign(row)">改派</el-button>
            <el-button v-permission="PERM.REPAIR_MANAGE" link type="danger" @click="handleDelete(row)">删除</el-button>
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

    <RepairFormDialog v-model="formVisible" :model="editing" @saved="handleSaved" />
    <FinishRepairDialog v-model="finishVisible" :model="finishing" @saved="handleSaved" />
    <FaultDetailDrawer v-model="detailVisible" :fault-id="activeFaultId" />
    <AssigneePickerDialog
      v-model="reassignVisible"
      title="改派维修负责人"
      :current-name="reassignTarget?.repairman"
      @confirm="handleReassign"
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import AssigneePickerDialog from '@/components/common/AssigneePickerDialog.vue'
import RepairFormDialog from './components/RepairFormDialog.vue'
import FinishRepairDialog from './components/FinishRepairDialog.vue'
import FaultDetailDrawer from '@/views/fault/components/FaultDetailDrawer.vue'
import { repairApi } from '@/api/repair'
import { exportApi } from '@/api/export'
import { useDictStore } from '@/stores/dict'
import { useAuthStore } from '@/stores/auth'
import { PERMISSIONS as PERM } from '@/constants/permission'
import { REPAIR_RESULT, REPAIR_STATUS } from '@/constants/dict'
import { formatDateTime, formatDuration, formatMoney } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const route = useRoute()
const router = useRouter()
const dictStore = useDictStore()
const auth = useAuthStore()

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(repairApi.list, {
  keyword: '',
  status: '',
  result: '',
  repairman: '',
  start_date: '',
  end_date: '',
})

const dateRange = ref([])
const formVisible = ref(false)
const finishVisible = ref(false)
const detailVisible = ref(false)
const reassignVisible = ref(false)
const editing = ref(null)
const finishing = ref(null)
const activeFaultId = ref(null)
const reassignTarget = ref(null)

// 是否可推进该记录: 管理岗可推进任意记录; 维修人员只能推进本人负责的记录。
function isMine(row) {
  return auth.user && row.assignee_id === auth.user.id
}
function canAdvance(row) {
  return auth.can(PERM.REPAIR_MANAGE) || isMine(row)
}

function openReassign(row) {
  reassignTarget.value = { ...row }
  reassignVisible.value = true
}

async function handleReassign(assigneeId) {
  try {
    await repairApi.reassign(reassignTarget.value.id, { assignee_id: assigneeId })
    ElMessage.success('已改派维修负责人')
    reassignVisible.value = false
    load()
  } catch {
    // 提示由拦截器处理
  }
}

async function handleExport() {
  try {
    await exportApi.repairs({
      keyword: query.keyword,
      status: query.status,
      result: query.result,
      repairman: query.repairman,
      start_date: query.start_date,
      end_date: query.end_date,
    })
    ElMessage.success('导出已开始')
  } catch {
    // 越权提示由拦截器处理
  }
}

function applyDateRange() {
  query.start_date = dateRange.value?.[0] ?? ''
  query.end_date = dateRange.value?.[1] ?? ''
}

function handleSearch() {
  applyDateRange()
  search()
}

function handleReset() {
  dateRange.value = []
  reset()
}

function openCreate() {
  editing.value = null
  formVisible.value = true
}

function openEdit(row) {
  editing.value = { ...row }
  formVisible.value = true
}

function openFinish(row) {
  finishing.value = { ...row }
  finishVisible.value = true
}

function openDetail(row) {
  activeFaultId.value = row.fault_id
  detailVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除维修记录 ${row.repair_no} ?`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }

  try {
    await repairApi.remove(row.id)
    ElMessage.success('维修记录已删除')
    handleSaved()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleSaved() {
  load()
  dictStore.loadRepairMeta().catch(() => {})
}

// 支持从其它页面携带 fault_id 直接录入维修记录。
async function applyRouteQuery() {
  const faultId = Number(route.query.fault_id)
  if (!faultId) return
  editing.value = null
  formVisible.value = true
  router.replace({ path: '/repairs' })
}

onMounted(async () => {
  dictStore.ensureLoaded().catch(() => {})
  await applyRouteQuery()
})
</script>
