<template>
  <div class="page-header">
    <div>
      <div class="page-header__title">{{ title }}</div>
      <div v-if="description" class="page-header__desc text-muted">{{ description }}</div>
    </div>
    <div class="page-header__actions">
      <slot />
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.page-header__title {
  font-size: 18px;
  font-weight: 600;
}

.page-header__desc {
  font-size: 13px;
  margin-top: 4px;
}

.page-header__actions {
  display: flex;
  gap: 8px;
}
</style>
