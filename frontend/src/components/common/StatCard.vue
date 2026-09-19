<template>
  <el-card shadow="hover" class="stat-card">
    <div class="stat-card__body">
      <div>
        <div class="stat-card__label text-muted">{{ label }}</div>
        <div class="stat-card__value" :style="{ color }">
          {{ value }}<span v-if="suffix" class="stat-card__suffix">{{ suffix }}</span>
        </div>
        <div v-if="hint" class="stat-card__hint text-muted">{{ hint }}</div>
      </div>
      <el-icon v-if="icon" :size="34" :color="color" class="stat-card__icon">
        <component :is="icon" />
      </el-icon>
    </div>
  </el-card>
</template>

<script setup>
defineProps({
  label: { type: String, required: true },
  value: { type: [String, Number], default: 0 },
  suffix: { type: String, default: '' },
  hint: { type: String, default: '' },
  icon: { type: String, default: '' },
  color: { type: String, default: '#303133' },
})
</script>

<style scoped>
.stat-card__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stat-card__label {
  font-size: 13px;
}

.stat-card__value {
  font-size: 26px;
  font-weight: 600;
  margin-top: 6px;
}

.stat-card__suffix {
  font-size: 13px;
  font-weight: 400;
  margin-left: 4px;
}

.stat-card__hint {
  font-size: 12px;
  margin-top: 4px;
}
</style>
