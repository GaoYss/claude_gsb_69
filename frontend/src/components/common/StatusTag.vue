<template>
  <el-tag :type="dictType(dict, value)" size="small" effect="light">
    {{ dictLabel(dict, value) }}
  </el-tag>
</template>

<script setup>
import { dictLabel, dictType } from '@/constants/dict'

defineProps({
  dict: { type: Object, required: true },
  value: { type: String, default: '' },
})
</script>
