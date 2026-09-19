<template>
  <div class="bar-list">
    <div v-for="item in items" :key="item.label" class="bar-item">
      <div class="bar-item__header">
        <span>{{ item.label }}</span>
        <span class="text-muted">{{ item.count }}</span>
      </div>
      <el-progress :percentage="percent(item.count)" :show-text="false" :stroke-width="10" />
    </div>
    <el-empty v-if="!items.length" description="暂无数据" :image-size="60" />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
})

const max = computed(() => Math.max(1, ...props.items.map((item) => Number(item.count) || 0)))

function percent(count) {
  return Math.round(((Number(count) || 0) / max.value) * 100)
}
</script>
