<template>
  <div class="page">
    <PageHeader title="维修进度追踪" description="按故障单号或路灯编号查看从登记、开工、完工到闭环的完整链路" />

    <el-card shadow="never">
      <div class="filter-bar">
        <el-radio-group v-model="searchType">
          <el-radio-button value="fault_no">按故障单号</el-radio-button>
          <el-radio-button value="lamp_code">按路灯编号</el-radio-button>
        </el-radio-group>
        <el-input
          v-model="keyword"
          :placeholder="searchType === 'fault_no' ? '例如 GD202609130001' : '例如 LD-00001'"
          clearable
          style="width: 260px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" :loading="loading" @click="handleSearch">查询</el-button>
      </div>
    </el-card>

    <template v-if="result">
      <el-row :gutter="16">
        <el-col :xs="24" :md="14">
          <el-card v-if="result.fault" shadow="never">
            <div class="section-title">
              <span>故障信息</span>
              <StatusTag :dict="FAULT_STATUS" :value="result.fault.status" />
            </div>
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="故障单号">{{ result.fault.fault_no }}</el-descriptions-item>
              <el-descriptions-item label="故障类型">{{ result.fault.fault_type }}</el-descriptions-item>
              <el-descriptions-item label="紧急程度">
                <StatusTag :dict="FAULT_LEVEL" :value="result.fault.fault_level" />
              </el-descriptions-item>
              <el-descriptions-item label="故障来源">{{ dictLabel(FAULT_SOURCE, result.fault.source) }}</el-descriptions-item>
              <el-descriptions-item label="上报人">{{ result.fault.reporter || '-' }}</el-descriptions-item>
              <el-descriptions-item label="上报时间">{{ formatDateTime(result.fault.reported_at) }}</el-descriptions-item>
              <el-descriptions-item label="维修次数">{{ result.fault.repair_count }} 次</el-descriptions-item>
              <el-descriptions-item label="当前耗时">{{ elapsedText }}</el-descriptions-item>
              <el-descriptions-item label="故障描述" :span="2">{{ result.fault.description || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-empty v-else description="该路灯暂无故障记录" />
        </el-col>
        <el-col :xs="24" :md="10">
          <el-card v-if="result.lamp" shadow="never">
            <div class="section-title">路灯信息</div>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="路灯编号">{{ result.lamp.code }}</el-descriptions-item>
              <el-descriptions-item label="名称">{{ result.lamp.name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="所在道路">{{ result.lamp.road_name }}</el-descriptions-item>
              <el-descriptions-item label="运行状态">
                <StatusTag :dict="RUN_STATUS" :value="result.lamp.run_status" />
              </el-descriptions-item>
              <el-descriptions-item label="灯具类型">{{ result.lamp.lamp_type || '-' }}</el-descriptions-item>
              <el-descriptions-item label="安装日期">{{ formatDate(result.lamp.install_date) }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never">
        <div class="section-title">处置时间线</div>
        <el-timeline v-if="result.timeline?.length">
          <el-timeline-item
            v-for="(event, index) in result.timeline"
            :key="index"
            :timestamp="formatDateTime(event.timestamp)"
            :type="dictType(TIMELINE_STAGE, event.stage)"
          >
            <div class="timeline-title">{{ event.label }}</div>
            <div class="text-muted">{{ event.operator || '系统' }} · {{ event.detail || '-' }}</div>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-else description="暂无处置记录" :image-size="80" />
      </el-card>

      <el-card shadow="never">
        <div class="section-title">维修记录明细</div>
        <el-table :data="result.repairs" size="small" border>
          <el-table-column prop="repair_no" label="维修单号" width="150" />
          <el-table-column prop="repairman" label="维修人员" width="110" />
          <el-table-column prop="repair_team" label="班组" width="140" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><StatusTag :dict="REPAIR_STATUS" :value="row.status" /></template>
          </el-table-column>
          <el-table-column label="结果" width="100">
            <template #default="{ row }">{{ dictLabel(REPAIR_RESULT, row.result) }}</template>
          </el-table-column>
          <el-table-column label="开工" width="150">
            <template #default="{ row }">{{ formatDateTime(row.started_at) }}</template>
          </el-table-column>
          <el-table-column label="完工" width="150">
            <template #default="{ row }">{{ formatDateTime(row.finished_at) }}</template>
          </el-table-column>
          <el-table-column label="耗时" width="120">
            <template #default="{ row }">{{ formatDuration(row.duration_minutes) }}</template>
          </el-table-column>
          <el-table-column prop="materials" label="耗材" min-width="140" show-overflow-tooltip />
          <el-table-column label="费用" width="100">
            <template #default="{ row }">{{ formatMoney(row.cost) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card v-if="result.related_faults?.length" shadow="never">
        <div class="section-title">该路灯的历史故障</div>
        <el-table :data="result.related_faults" size="small">
          <el-table-column prop="fault_no" label="故障单号" width="150" />
          <el-table-column prop="fault_type" label="类型" width="110" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><StatusTag :dict="FAULT_STATUS" :value="row.status" /></template>
          </el-table-column>
          <el-table-column label="上报时间" width="160">
            <template #default="{ row }">{{ formatDateTime(row.reported_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="primary" @click="goFault(row.fault_no)">查看链路</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>

    <el-card v-else shadow="never">
      <el-empty description="请输入故障单号或路灯编号后查询" />
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import { statusApi } from '@/api/status'
import {
  FAULT_LEVEL,
  FAULT_SOURCE,
  FAULT_STATUS,
  REPAIR_RESULT,
  REPAIR_STATUS,
  RUN_STATUS,
  TIMELINE_STAGE,
  dictLabel,
  dictType,
} from '@/constants/dict'
import { formatDate, formatDateTime, formatDuration, formatMoney, formatWaiting } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const searchType = ref('fault_no')
const keyword = ref('')
const loading = ref(false)
const result = ref(null)

// 已闭环的故障展示"上报 -> 闭环"的总耗时, 未闭环的展示从上报至今的等待时长
const elapsedText = computed(() => {
  const fault = result.value?.fault
  if (!fault?.reported_at) return '-'
  const start = new Date(fault.reported_at).getTime()
  const end = fault.closed_at ? new Date(fault.closed_at).getTime() : Date.now()
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) return '-'
  return formatWaiting((end - start) / 3600000)
})

async function query(params) {
  loading.value = true
  try {
    result.value = await statusApi.track(params)
  } catch (error) {
    result.value = null
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  const value = keyword.value.trim()
  if (!value) {
    result.value = null
    return
  }
  query({ [searchType.value]: value })
}

function goFault(faultNo) {
  searchType.value = 'fault_no'
  keyword.value = faultNo
  query({ fault_no: faultNo })
}

// 支持从其它页面通过 query 参数直接进入追踪结果。
onMounted(() => {
  const faultNo = route.query.fault_no
  const lampCode = route.query.lamp_code
  if (faultNo) {
    searchType.value = 'fault_no'
    keyword.value = String(faultNo)
    query({ fault_no: String(faultNo) })
    return
  }
  if (lampCode) {
    searchType.value = 'lamp_code'
    keyword.value = String(lampCode)
    query({ lamp_code: String(lampCode) })
  }
})
</script>

<style scoped>
.timeline-title {
  font-weight: 600;
}
</style>
