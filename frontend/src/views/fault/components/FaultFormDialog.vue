<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑故障登记' : '登记路灯故障'"
    width="680px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="所属路灯" prop="lamp_id">
        <el-select
          v-model="form.lamp_id"
          filterable
          remote
          reserve-keyword
          :disabled="isEdit"
          :remote-method="searchLamps"
          :loading="lampLoading"
          placeholder="输入路灯编号 / 道路名称搜索"
          style="width: 100%"
        >
          <el-option
            v-for="item in lampCandidates"
            :key="item.id"
            :label="`${item.code} · ${item.road_name} · ${item.name || '未命名'}`"
            :value="item.id"
          />
        </el-select>
        <div v-if="selectedLamp" class="form-hint text-muted">
          当前状态: {{ runStatusText }} · {{ selectedLamp.lamp_type }} · {{ selectedLamp.power || 0 }} W
        </div>
      </el-form-item>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="故障类型" prop="fault_type">
            <el-select v-model="form.fault_type" placeholder="请选择故障类型" style="width: 100%">
              <el-option v-for="item in faultTypeOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="紧急程度" prop="fault_level">
            <el-select v-model="form.fault_level" style="width: 100%">
              <el-option v-for="(item, key) in FAULT_LEVEL" :key="key" :label="item.label" :value="key" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="故障来源" prop="source">
            <el-select v-model="form.source" style="width: 100%">
              <el-option v-for="(item, key) in FAULT_SOURCE" :key="key" :label="item.label" :value="key" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="上报时间" prop="reported_at">
            <el-date-picker
              v-model="form.reported_at"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="默认取当前时间"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="上报人" prop="reporter">
            <el-input v-model="form.reporter" placeholder="例如 巡检员王师傅" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系电话" prop="reporter_phone">
            <el-input v-model="form.reporter_phone" placeholder="选填" />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="故障描述" prop="description">
            <el-input
              v-model="form.description"
              type="textarea"
              :rows="3"
              maxlength="512"
              show-word-limit
              placeholder="请描述现场现象, 例如: 整灯不亮、灯光闪烁、灯杆倾斜等"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { faultApi } from '@/api/fault'
import { lampApi } from '@/api/lamp'
import { FAULT_LEVEL, FAULT_SOURCE, RUN_STATUS, dictLabel } from '@/constants/dict'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
  presetLamp: { type: Object, default: null },
  faultTypeOptions: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)
const lampLoading = ref(false)
const lampCandidates = ref([])
const cachedLamp = ref(null)

const isEdit = computed(() => Boolean(props.model?.id))
const selectedLamp = computed(() => cachedLamp.value)
const runStatusText = computed(() => dictLabel(RUN_STATUS, selectedLamp.value?.run_status))

const createForm = () => ({
  lamp_id: undefined,
  fault_type: '',
  fault_level: 'normal',
  source: 'inspection',
  description: '',
  reporter: '',
  reporter_phone: '',
  reported_at: '',
})

const form = reactive(createForm())

const rules = {
  lamp_id: [{ required: true, message: '请选择所属路灯', trigger: 'change' }],
  fault_type: [{ required: true, message: '请选择故障类型', trigger: 'change' }],
  description: [{ required: true, message: '请填写故障描述', trigger: 'blur' }],
}

async function searchLamps(keyword = '') {
  lampLoading.value = true
  try {
    const data = await lampApi.list({ keyword, page: 1, page_size: 20 }, { silent: true })
    lampCandidates.value = data?.items ?? []
  } catch (error) {
    lampCandidates.value = []
  } finally {
    lampLoading.value = false
  }
}

// 打开弹窗时初始化表单: 编辑模式回填数据, 新增模式支持从台账页带入路灯。
async function syncForm() {
  Object.assign(form, createForm())
  cachedLamp.value = null

  if (props.model) {
    Object.assign(form, {
      lamp_id: props.model.lamp_id,
      fault_type: props.model.fault_type,
      fault_level: props.model.fault_level,
      source: props.model.source,
      description: props.model.description,
      reporter: props.model.reporter,
      reporter_phone: props.model.reporter_phone,
      reported_at: props.model.reported_at ? props.model.reported_at.replace('T', ' ').slice(0, 19) : '',
    })
    try {
      cachedLamp.value = await lampApi.detail(props.model.lamp_id, { silent: true })
      lampCandidates.value = cachedLamp.value ? [cachedLamp.value] : []
    } catch (error) {
      lampCandidates.value = []
    }
    return
  }

  await searchLamps('')
  if (props.presetLamp) {
    cachedLamp.value = props.presetLamp
    form.lamp_id = props.presetLamp.id
    if (!lampCandidates.value.some((item) => item.id === props.presetLamp.id)) {
      lampCandidates.value = [props.presetLamp, ...lampCandidates.value]
    }
  }
}

function handleLampChange(id) {
  cachedLamp.value = lampCandidates.value.find((item) => item.id === id) ?? null
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const payload = { ...form }
    if (!payload.reported_at) {
      delete payload.reported_at
    }
    if (isEdit.value) {
      const { lamp_id: _ignored, ...rest } = payload
      await faultApi.update(props.model.id, rest)
      ElMessage.success('故障登记信息已更新')
    } else {
      await faultApi.create(payload)
      ElMessage.success('故障登记成功, 路灯状态已更新为故障')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.form-hint {
  font-size: 12px;
  line-height: 1.6;
}
</style>
