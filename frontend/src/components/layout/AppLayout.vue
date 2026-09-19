<template>
  <el-container class="app-layout">
    <el-aside width="220px" class="app-aside">
      <div class="app-brand">
        <el-icon :size="20"><Sunny /></el-icon>
        <span>路灯故障登记系统</span>
      </div>
      <el-menu :default-active="activeMenu" router class="app-menu">
        <el-menu-item v-for="item in menus" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="app-header">
        <div class="app-header__title">{{ currentTitle }}</div>
        <div class="app-header__extra">
          <el-tag type="success" effect="plain">{{ today }}</el-tag>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { formatDate } from '@/utils/format'

const route = useRoute()

const menus = [
  { path: '/dashboard', title: '运行看板', icon: 'DataLine' },
  { path: '/lamps', title: '路灯台账', icon: 'Postcard' },
  { path: '/faults', title: '故障登记', icon: 'Warning' },
  { path: '/repairs', title: '维修记录录入', icon: 'Tools' },
  { path: '/status', title: '维修状态查询', icon: 'Search' },
  { path: '/status/track', title: '维修进度追踪', icon: 'Guide' },
]

const activeMenu = computed(() => {
  const matched = menus
    .filter((item) => route.path === item.path || route.path.startsWith(`${item.path}/`))
    .sort((a, b) => b.path.length - a.path.length)
  return matched[0]?.path ?? route.path
})

const currentTitle = computed(() => route.meta?.title ?? '路灯故障登记系统')
const today = computed(() => formatDate(new Date()))
</script>

<style scoped>
.app-layout {
  height: 100vh;
}

.app-aside {
  background-color: #1f2d3d;
  display: flex;
  flex-direction: column;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 60px;
  padding: 0 16px;
  color: #fff;
  font-weight: 600;
  font-size: 15px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.app-menu {
  flex: 1;
  border-right: none;
  background-color: transparent;
}

.app-menu :deep(.el-menu-item) {
  color: #c0c4cc;
}

.app-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background-color: #409eff;
}

.app-menu :deep(.el-menu-item:hover) {
  background-color: rgba(64, 158, 255, 0.2);
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  border-bottom: 1px solid var(--app-border);
}

.app-header__title {
  font-size: 16px;
  font-weight: 600;
}

.app-main {
  background-color: var(--app-bg);
  padding: 16px;
}
</style>
