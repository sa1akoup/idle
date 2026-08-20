<!-- 地图页：使用城区视觉底图展示节点、距离、物资与连接关系。 -->
<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { Clock, Connection, Location, Warning } from '@element-plus/icons-vue'
import type { Enemy, GameMap, MapNode } from '../types'

const props = defineProps<{
  maps: GameMap[]
  nodes: MapNode[]
  enemies: Enemy[]
}>()

const selectedNodeId = ref('')

const positions: Record<string, { left: string; top: string }> = {
  node_apt: { left: '13%', top: '19%' },
  node_warehouse: { left: '39%', top: '25%' },
  node_customs: { left: '70%', top: '18%' },
  node_tunnel: { left: '23%', top: '70%' },
  node_container: { left: '55%', top: '62%' },
  node_pier: { left: '83%', top: '76%' },
}

watchEffect(() => {
  if (!props.nodes.some((item) => item.id === selectedNodeId.value)) selectedNodeId.value = props.nodes[0]?.id ?? ''
})

const selectedNode = computed(() => props.nodes.find((item) => item.id === selectedNodeId.value))
const selectedEnemy = computed(() => props.enemies.find((item) => item.id === selectedNode.value?.enemyId))
const distanceName = computed(() => ({ close: '近距离', mid: '中距离', far: '远距离' })[selectedNode.value?.distance ?? 'mid'])
const extractionNodeId = computed(() => props.maps[0]?.extractionNodeId)
const orderedNodes = computed(() => [...props.nodes].sort((a, b) => a.routeOrder - b.routeOrder))
const nextNodeName = computed(() => {
  const nextID = selectedNode.value?.connections.split(',').map((item) => item.trim()).find(Boolean)
  return orderedNodes.value.find((item) => item.id === nextID)?.name || (nextID ? nextID : '路线终点')
})
</script>

<template>
  <section class="view-page map-view">
    <header class="page-heading">
      <div><span class="eyebrow">区域情报</span><h1>地图</h1><p>{{ maps[0]?.desc || '区域情报载入中' }}</p></div>
      <span class="intel-stamp">INTEL / 01</span>
    </header>

    <div class="map-layout">
      <div class="tactical-map" aria-label="废弃城区节点地图">
        <img src="/city-map.svg" alt="废弃城区俯视战术底图" />
        <button
          v-for="(node, index) in orderedNodes"
          :key="node.id"
          type="button"
          class="map-node"
          :class="{ active: node.id === selectedNodeId, extract: node.id === extractionNodeId }"
          :style="positions[node.id]"
          @click="selectedNodeId = node.id"
        >
          <span>{{ String(node.routeOrder || index + 1).padStart(2, '0') }}</span>{{ node.name }}
        </button>
      </div>

      <aside v-if="selectedNode" class="node-inspector surface-panel">
        <div class="panel-heading"><div><span>NODE</span><h2>{{ selectedNode.name }}</h2></div><el-icon><Location /></el-icon></div>
        <dl class="intel-list">
          <div><dt><el-icon><Clock /></el-icon>探索耗时</dt><dd>{{ selectedNode.exploreTime }} 分钟</dd></div>
          <div><dt><el-icon><Warning /></el-icon>交战距离</dt><dd>{{ distanceName }}</dd></div>
          <div><dt><el-icon><Connection /></el-icon>下一站</dt><dd>{{ nextNodeName }}</dd></div>
          <div><dt><el-icon><Location /></el-icon>节点价值</dt><dd>V{{ selectedNode.valueTier }} · 第{{ selectedNode.routeOrder }}站</dd></div>
        </dl>
        <div class="node-detail"><span>容器池 · 槽位 {{ selectedNode.containerSlots }}</span><div v-if="selectedNode.containers.length" class="loot-list"><span v-for="container in selectedNode.containers" :key="`${container.pool}-${container.id}`" class="loot-chip" :title="`${container.pool} · ${container.tags.join(' / ')} · 搜索风险 ${container.searchRisk} · 耗时 ${container.searchTime} 分钟`">{{ container.pool }} · {{ container.name }} · V{{ container.valueTier }} · W{{ container.weight }}</span></div><p v-else>暂无容器</p></div>
        <div class="node-detail"><span>主要威胁</span><p>{{ selectedEnemy?.name || '未知活动' }} · HP {{ selectedEnemy?.hp || '--' }}</p></div>
        <div class="node-detail"><span>单向路线</span><p>{{ nextNodeName }}</p></div>
      </aside>
    </div>
  </section>
</template>
