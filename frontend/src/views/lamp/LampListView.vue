<template>
  <div class="page">
    <PageHeader title="路灯台账" description="维护每盏路灯的档案信息与运行状态, 台账是故障登记与维修的基础数据">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增路灯</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="编号 / 名称 / 道路 / 地址" clearable @keyup.enter="search" />
        <el-select v-model="query.road_name" placeholder="所在道路" clearable>
          <el-option v-for="road in dictStore.lampOptions.roads" :key="road" :label="road" :value="road" />
        </el-select>
        <el-select v-model="query.lamp_type" placeholder="灯具类型" clearable>
          <el-option v-for="item in dictStore.lampOptions.lamp_types" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="query.run_status" placeholder="运行状态" clearable>
          <el-option v-for="(item, key) in RUN_STATUS" :key="key" :label="item.label" :value="key" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="search">查询</el-button>
        <el-button :icon="RefreshLeft" @click="reset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="code" label="路灯编号" width="120" fixed="left" />
        <el-table-column prop="name" label="名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="road_name" label="所在道路" min-width="120" show-overflow-tooltip />
        <el-table-column prop="district" label="区域" width="100" />
        <el-table-column prop="lamp_type" label="灯具类型" width="110" />
        <el-table-column label="功率" width="90">
          <template #default="{ row }">{{ row.power ? `${row.power} W` : '-' }}</template>
        </el-table-column>
        <el-table-column label="灯杆高度" width="100">
          <template #default="{ row }">{{ row.pole_height ? `${row.pole_height} m` : '-' }}</template>
        </el-table-column>
        <el-table-column label="运行状态" width="100">
          <template #default="{ row }"><StatusTag :dict="RUN_STATUS" :value="row.run_status" /></template>
        </el-table-column>
        <el-table-column label="安装日期" width="120">
          <template #default="{ row }">{{ formatDate(row.install_date) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" @click="goRegisterFault(row)">登记故障</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
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

    <LampFormDialog
      v-model="dialogVisible"
      :model="editing"
      :next-code="dictStore.lampOptions.next_code"
      :road-options="dictStore.lampOptions.roads"
      :district-options="dictStore.lampOptions.districts"
      :lamp-type-options="dictStore.lampOptions.lamp_types"
      @saved="handleSaved"
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import LampFormDialog from './components/LampFormDialog.vue'
import { lampApi } from '@/api/lamp'
import { useDictStore } from '@/stores/dict'
import { RUN_STATUS } from '@/constants/dict'
import { formatDate } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const router = useRouter()
const dictStore = useDictStore()

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(
  lampApi.list,
  { keyword: '', road_name: '', lamp_type: '', run_status: '' },
)

const dialogVisible = ref(false)
const editing = ref(null)

function openCreate() {
  editing.value = null
  dialogVisible.value = true
}

function openEdit(row) {
  editing.value = { ...row }
  dialogVisible.value = true
}

function goRegisterFault(row) {
  router.push({ path: '/faults', query: { lamp_id: row.id, lamp_code: row.code } })
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除路灯 ${row.code} ? 存在未闭环故障时无法删除。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }

  try {
    await lampApi.remove(row.id)
    ElMessage.success('路灯已删除')
    load()
    dictStore.loadLampOptions()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleSaved() {
  load()
  dictStore.loadLampOptions()
}

onMounted(() => {
  dictStore.ensureLoaded().catch(() => {})
})
</script>
