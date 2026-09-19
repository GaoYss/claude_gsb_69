<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑路灯台账' : '新增路灯台账'"
    width="720px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="路灯编号" prop="code">
            <el-input v-model="form.code" placeholder="例如 LD-00001">
              <template v-if="!isEdit && nextCode" #append>
                <el-button @click="form.code = nextCode">用建议值</el-button>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="路灯名称" prop="name">
            <el-input v-model="form.name" placeholder="例如 中山路3号灯杆" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="所在道路" prop="road_name">
            <el-select v-model="form.road_name" filterable allow-create placeholder="选择或输入道路名称" style="width: 100%">
              <el-option v-for="road in roadOptions" :key="road" :label="road" :value="road" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="所属区域" prop="district">
            <el-select v-model="form.district" filterable allow-create clearable placeholder="选择或输入区域" style="width: 100%">
              <el-option v-for="item in districtOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="安装位置" prop="address">
            <el-input v-model="form.address" placeholder="例如 中山路200号门前" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="灯具类型" prop="lamp_type">
            <el-select v-model="form.lamp_type" placeholder="请选择灯具类型" style="width: 100%">
              <el-option v-for="item in lampTypeOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="额定功率" prop="power">
            <el-input-number v-model="form.power" :min="0" :max="3000" :step="10" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="灯杆高度" prop="pole_height">
            <el-input-number v-model="form.pole_height" :min="0" :max="100" :step="0.5" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="安装日期" prop="install_date">
            <el-date-picker v-model="form.install_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="经度" prop="longitude">
            <el-input-number v-model="form.longitude" :min="-180" :max="180" :precision="6" :step="0.0001" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="纬度" prop="latitude">
            <el-input-number v-model="form.latitude" :min="-90" :max="90" :precision="6" :step="0.0001" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="运行状态" prop="run_status">
            <el-select v-model="form.run_status" style="width: 100%">
              <el-option v-for="(item, key) in RUN_STATUS" :key="key" :label="item.label" :value="key" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="备注" prop="remark">
            <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit />
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
import { lampApi } from '@/api/lamp'
import { RUN_STATUS } from '@/constants/dict'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
  nextCode: { type: String, default: '' },
  roadOptions: { type: Array, default: () => [] },
  districtOptions: { type: Array, default: () => [] },
  lampTypeOptions: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)
const isEdit = computed(() => Boolean(props.model?.id))

const createForm = () => ({
  code: '',
  name: '',
  road_name: '',
  district: '',
  address: '',
  lamp_type: 'LED',
  power: 100,
  pole_height: 8,
  longitude: undefined,
  latitude: undefined,
  run_status: 'normal',
  install_date: '',
  remark: '',
})

const form = reactive(createForm())

const rules = {
  code: [{ required: true, message: '请输入路灯编号', trigger: 'blur' }],
  road_name: [{ required: true, message: '请选择或输入所在道路', trigger: 'change' }],
  lamp_type: [{ required: true, message: '请选择灯具类型', trigger: 'change' }],
}

// 每次打开弹窗时同步表单数据, 区分新增与编辑。
function syncForm() {
  Object.assign(form, createForm())
  if (props.model) {
    Object.assign(form, {
      ...props.model,
      install_date: props.model.install_date ? String(props.model.install_date).slice(0, 10) : '',
    })
  }
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await lampApi.update(props.model.id, { ...form })
      ElMessage.success('路灯台账已更新')
    } else {
      await lampApi.create({ ...form })
      ElMessage.success('路灯台账已创建')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>
