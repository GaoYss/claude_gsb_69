<template>
  <el-dialog
    :model-value="modelValue"
    title="完成维修"
    width="600px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-descriptions v-if="model" :column="2" border size="small" class="repair-summary">
      <el-descriptions-item label="维修单号">{{ model.repair_no }}</el-descriptions-item>
      <el-descriptions-item label="故障单号">{{ model.fault_no }}</el-descriptions-item>
      <el-descriptions-item label="维修人员">{{ model.repairman }}</el-descriptions-item>
      <el-descriptions-item label="开工时间">{{ formatDateTime(model.started_at) }}</el-descriptions-item>
    </el-descriptions>

    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="维修结果" prop="result">
        <el-select v-model="form.result" style="width: 100%">
          <el-option v-for="(item, key) in REPAIR_RESULT" :key="key" :label="item.label" :value="key" />
        </el-select>
        <div class="form-hint text-muted">选择"已修复"后, 故障将自动流转为已修复, 路灯恢复为正常状态。</div>
      </el-form-item>
      <el-form-item label="完工时间" prop="finished_at">
        <el-date-picker
          v-model="form.finished_at"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          placeholder="默认取当前时间"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="维修内容" prop="content">
        <el-input v-model="form.content" type="textarea" :rows="2" maxlength="512" show-word-limit />
      </el-form-item>
      <el-form-item label="使用耗材" prop="materials">
        <el-input v-model="form.materials" placeholder="例如: 驱动电源 1 个" />
      </el-form-item>
      <el-form-item label="费用(元)" prop="cost">
        <el-input-number v-model="form.cost" :min="0" :precision="2" :step="10" style="width: 100%" />
      </el-form-item>
      <el-form-item label="备注" prop="remark">
        <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">提交完成</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { repairApi } from '@/api/repair'
import { REPAIR_RESULT } from '@/constants/dict'
import { formatDateTime } from '@/utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)

const createForm = () => ({
  result: 'fixed',
  finished_at: '',
  content: '',
  materials: '',
  cost: 0,
  remark: '',
})

const form = reactive(createForm())

const rules = {
  result: [{ required: true, message: '请选择维修结果', trigger: 'change' }],
}

function syncForm() {
  Object.assign(form, createForm())
  if (props.model) {
    form.content = props.model.content ?? ''
    form.materials = props.model.materials ?? ''
    form.cost = Number(props.model.cost ?? 0)
  }
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const payload = { ...form }
    if (!payload.finished_at) {
      delete payload.finished_at
    }
    await repairApi.finish(props.model.id, payload)
    ElMessage.success('维修记录已完成')
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.repair-summary {
  margin-bottom: 16px;
}

.form-hint {
  font-size: 12px;
  line-height: 1.6;
}
</style>
