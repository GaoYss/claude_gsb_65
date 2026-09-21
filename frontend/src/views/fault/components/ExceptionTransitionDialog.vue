<template>
  <el-dialog
    :model-value="modelValue"
    title="例外流转（管理岗）"
    width="460px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert
      type="warning"
      :closable="false"
      title="例外流转会跳过常规状态机直接改变状态, 必须填写原因, 操作将写入审计日志。"
      class="warn-tip"
    />
    <el-form label-width="92px">
      <el-form-item label="当前状态">
        <el-tag>{{ currentStatusLabel }}</el-tag>
      </el-form-item>
      <el-form-item label="目标状态">
        <el-select v-model="toStatus" style="width: 100%">
          <el-option
            v-for="(item, key) in FAULT_STATUS"
            :key="key"
            :label="item.label"
            :value="key"
            :disabled="key === currentStatus"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="例外原因">
        <el-input v-model="reason" type="textarea" :rows="3" maxlength="255" show-word-limit placeholder="请说明为何需要跳过标准流程" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="danger" @click="confirm">确认例外流转</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { FAULT_STATUS, dictLabel } from '@/constants/dict'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  currentStatus: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'confirm'])

const toStatus = ref('')
const reason = ref('')

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      toStatus.value = ''
      reason.value = ''
    }
  },
)

const currentStatusLabel = () => dictLabel(FAULT_STATUS, props.currentStatus)

function confirm() {
  if (!toStatus.value) {
    ElMessage.warning('请选择目标状态')
    return
  }
  if (!reason.value.trim()) {
    ElMessage.warning('必须填写例外原因')
    return
  }
  emit('confirm', { to_status: toStatus.value, reason: reason.value.trim() })
}
</script>

<style scoped>
.warn-tip {
  margin-bottom: 16px;
}
</style>
