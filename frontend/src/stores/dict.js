import { ref } from 'vue'
import { defineStore } from 'pinia'
import { lampApi } from '@/api/lamp'
import { faultApi } from '@/api/fault'
import { repairApi } from '@/api/repair'

// 字典数据在多个页面复用, 统一缓存避免重复请求。
export const useDictStore = defineStore('dict', () => {
  const lampOptions = ref({
    roads: [],
    districts: [],
    lamp_types: [],
    run_statuses: [],
    next_code: '',
  })
  const faultMeta = ref({ statuses: [], levels: [], sources: [], fault_types: [] })
  const repairMeta = ref({ statuses: [], results: [], repairmen: [], teams: [] })
  const loaded = ref(false)

  async function loadLampOptions() {
    lampOptions.value = await lampApi.options()
  }

  async function loadFaultMeta() {
    faultMeta.value = await faultApi.meta()
  }

  async function loadRepairMeta() {
    repairMeta.value = await repairApi.meta()
  }

  async function refreshAll() {
    await Promise.all([loadLampOptions(), loadFaultMeta(), loadRepairMeta()])
    loaded.value = true
  }

  async function ensureLoaded() {
    if (loaded.value) return
    await refreshAll()
  }

  return {
    lampOptions,
    faultMeta,
    repairMeta,
    loaded,
    loadLampOptions,
    loadFaultMeta,
    loadRepairMeta,
    refreshAll,
    ensureLoaded,
  }
})
