<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="440px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="loadRepairmen"
  >
    <el-form label-width="92px">
      <el-form-item label="当前负责人">
        <span>{{ currentName || '未指派' }}</span>
      </el-form-item>
      <el-form-item label="改派给">
        <el-select v-model="assigneeId" placeholder="选择维修人员" style="width: 100%" :loading="loading">
          <el-option
            v-for="item in repairmen"
            :key="item.id"
            :label="`${item.display_name}（${item.username}）`"
            :value="item.id"
          />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="confirm">确认改派</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { userApi } from '@/api/auth'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '改派负责人' },
  currentName: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'confirm'])

const loading = ref(false)
const repairmen = ref([])
const assigneeId = ref(null)

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      assigneeId.value = null
      loadRepairmen()
    }
  },
)

async function loadRepairmen() {
  loading.value = true
  try {
    // 改派仅管理岗可用, 用户接口也仅管理岗可访问。
    const data = await userApi.list({ role: 'repair', page_size: 200 })
    repairmen.value = data.items || []
  } catch {
    repairmen.value = []
  } finally {
    loading.value = false
  }
}

function confirm() {
  if (!assigneeId.value) {
    ElMessage.warning('请选择改派目标')
    return
  }
  emit('confirm', assigneeId.value)
}
</script>
