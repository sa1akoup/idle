<!-- 游戏主导航：桌面固定侧栏，移动端使用可关闭抽屉。 -->
<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import {
  Box,
  Compass,
  Document,
  House,
  MapLocation,
  Operation,
  ShoppingCart,
  User,
  VideoPlay,
} from '@element-plus/icons-vue'
import type { NavKey, Player } from '../types'

const emit = defineEmits<{
  select: [key: NavKey]
  close: []
}>()

interface NavItem {
  key: NavKey
  label: string
  icon: Component
}

const props = defineProps<{
  active: NavKey
  mobileOpen: boolean
  player: Player | null
  cash: number
  hasActiveSession: boolean
}>()

const navigation = computed<NavItem[]>(() => {
  const items: NavItem[] = [
    { key: 'explore', label: '探索', icon: Compass },
    { key: 'map', label: '地图', icon: MapLocation },
    { key: 'character', label: '角色', icon: User },
    { key: 'inventory', label: '仓库', icon: Box },
    { key: 'merchant', label: '商人', icon: ShoppingCart },
    { key: 'hideout', label: '藏身处', icon: House },
  ]
  if (props.hasActiveSession) items.push({ key: 'live', label: '实时行动', icon: VideoPlay })
  items.push({ key: 'logs', label: '日志', icon: Document })
  return items
})

function select(key: NavKey) {
  emit('select', key)
  emit('close')
}
</script>

<template>
  <div v-if="mobileOpen" class="sidebar-backdrop" @click="emit('close')" />
  <aside class="sidebar" :class="{ 'is-open': mobileOpen }" aria-label="主功能导航">
    <div class="brand-block">
      <span class="brand-mark"><el-icon><Operation /></el-icon></span>
      <div>
        <strong>搜打撤</strong>
        <span>行动终端 / V0.2</span>
      </div>
    </div>

    <nav class="main-nav">
      <button
        v-for="item in navigation"
        :key="item.key"
        type="button"
        class="nav-item"
        :class="{ active: active === item.key }"
        :aria-current="active === item.key ? 'page' : undefined"
        @click="select(item.key)"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="sidebar-player">
      <span class="status-dot" />
      <div class="sidebar-player__info">
        <strong>{{ player?.name || '载入中' }}</strong>
        <span>{{ player && (player.hp <= 0 || player.energy <= 0 || player.hydration <= 0) ? '资源恢复中' : '可执行行动' }}</span>
      </div>
      <b>￥{{ cash.toLocaleString() }}</b>
    </div>
  </aside>
</template>
