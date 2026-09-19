import { onMounted, reactive, ref } from 'vue'

// 列表页通用逻辑: 分页查询、条件重置、翻页。
export function useListPage(fetcher, defaultQuery = {}, options = {}) {
  const { immediate = true } = options
  const loading = ref(false)
  const rows = ref([])
  const total = ref(0)
  const query = reactive({ page: 1, page_size: 10, ...defaultQuery })

  async function load() {
    loading.value = true
    try {
      const data = await fetcher({ ...query })
      rows.value = data?.items ?? []
      total.value = data?.total ?? 0
    } catch (error) {
      rows.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  function search() {
    query.page = 1
    return load()
  }

  function reset() {
    Object.assign(query, { page: 1, page_size: 10, ...defaultQuery })
    return load()
  }

  function changePage(page) {
    query.page = page
    return load()
  }

  function changePageSize(pageSize) {
    query.page = 1
    query.page_size = pageSize
    return load()
  }

  if (immediate) {
    onMounted(load)
  }

  return { loading, rows, total, query, load, search, reset, changePage, changePageSize }
}
