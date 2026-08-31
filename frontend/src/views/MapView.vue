<!-- 区域地图页：按 Graph API 数据展示九宫格节点、移动边与独立撤离点。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Clock, Connection, Location, Warning } from '@element-plus/icons-vue'
import MapGraphCanvas from '../components/MapGraphCanvas.vue'
import type { Enemy, GameMap, MapGraph } from '../types'

const props = defineProps<{
  maps: GameMap[]
  mapGraphs: Record<string, MapGraph>
  enemies: Enemy[]
}>()

const selectedNodeId = ref('')
const selectedMapId = computed(() => props.maps[0]?.id || '')
const graph = computed(() => props.mapGraphs[selectedMapId.value])

watch(graph, (value) => {
  if (!value?.nodes.some((node) => node.id === selectedNodeId.value)) selectedNodeId.value = value?.nodes[0]?.id || ''
}, { immediate: true })

const selectedNode = computed(() => graph.value?.nodes.find((node) => node.id === selectedNodeId.value))
const selectedEnemy = computed(() => props.enemies.find((enemy) => enemy.id === selectedNode.value?.enemyId))
const enemyThreat = computed(() => selectedEnemy.value ? `${selectedEnemy.value.name} · 阶层 ${['机体','杂鱼','守卫','精锐','BOSS'][selectedEnemy.value.kind === 'grunt' ? 1 : selectedEnemy.value.kind === 'guard' ? 2 : selectedEnemy.value.kind === 'elite' ? 3 : selectedEnemy.value.kind === 'boss' ? 4 : 1]} · HP 基准 ${selectedEnemy.value.hpBase}` : '未知活动')
const extractionPoints = computed(() => graph.value?.extractionPoints.filter((point) => point.enabled && point.anchorNodeId === selectedNodeId.value) || [])
const connectedNodes = computed(() => {
  if (!graph.value || !selectedNode.value) return []
  const ids = new Set<string>()
  for (const edge of graph.value.edges) {
    if (edge.fromNodeId === selectedNode.value.id) ids.add(edge.toNodeId)
    if (edge.bidirectional && edge.toNodeId === selectedNode.value.id) ids.add(edge.fromNodeId)
  }
  return graph.value.nodes.filter((node) => ids.has(node.id)).map((node) => node.name)
})
const distanceName = computed(() => ({ close: '近距离', mid: '中距离', far: '远距离' })[selectedNode.value?.distance || 'mid'])
</script>

<template>
  <section class="view-page map-view">
    <header class="page-heading">
      <div>
        <span class="eyebrow">区域情报</span>
        <h1>{{ graph?.map.name || '地图' }}</h1>
        <p>{{ graph?.map.desc || '地图图数据载入中' }}</p>
      </div>
      <span class="intel-stamp">GRAPH / {{ graph?.nodes.length || 0 }} NODES</span>
    </header>

    <div v-if="graph" class="map-layout">
      <div class="tactical-map">
        <MapGraphCanvas :graph="graph" :selected-node-id="selectedNodeId" @select="selectedNodeId = $event" />
        <div class="map-legend">
          <span><i class="legend-dot planned" />移动边</span>
          <span><i class="legend-dot extraction" />常规撤离</span>
        </div>
      </div>

      <aside v-if="selectedNode" class="node-inspector surface-panel">
        <div class="panel-heading"><div><span>NODE / {{ selectedNode.positionY * graph.map.layoutColumns + selectedNode.positionX + 1 }}</span><h2>{{ selectedNode.name }}</h2></div><el-icon><Location /></el-icon></div>
        <dl class="intel-list">
          <div><dt><el-icon><Clock /></el-icon>探索耗时</dt><dd>{{ selectedNode.exploreTime }} 分钟</dd></div>
          <div><dt><el-icon><Warning /></el-icon>交战距离</dt><dd>{{ distanceName }}</dd></div>
          <div><dt><el-icon><Connection /></el-icon>相邻节点</dt><dd>{{ connectedNodes.join('、') || '无' }}</dd></div>
          <div><dt><el-icon><Location /></el-icon>节点价值</dt><dd>V{{ selectedNode.valueTier }} · 槽位 {{ selectedNode.containerSlots }}</dd></div>
        </dl>

        <div v-if="extractionPoints.length" class="node-detail extraction-detail">
          <span>撤离点</span>
          <p v-for="point in extractionPoints" :key="point.id">{{ point.name }} · 锚点后 {{ point.travelTime }} 分钟抵达</p>
        </div>
        <div class="node-detail"><span>主要威胁</span><p>{{ enemyThreat }}</p></div>
        <div class="node-detail"><span>搜索容器</span><div v-if="selectedNode.containers.length" class="loot-list"><span v-for="container in selectedNode.containers" :key="`${container.pool}-${container.id}`" class="loot-chip" :title="`${container.pool} · ${container.tags.join(' / ')} · 搜索风险 ${container.searchRisk} · 耗时 ${container.searchTime} 分钟`">{{ container.pool }} · {{ container.name }} · V{{ container.valueTier }} · W{{ container.weight }}</span></div><p v-else>暂无容器</p></div>
      </aside>
    </div>
    <div v-else class="fatal-state"><strong>地图图数据暂不可用</strong><p>请刷新工作区后重试。</p></div>
  </section>
</template>

<style scoped>
.map-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
  gap: 18px;
  align-items: start;
}

.tactical-map {
  position: relative;
  min-width: 0;
}

.map-legend {
  display: flex;
  gap: 16px;
  align-items: center;
  padding: 10px 2px 0;
  color: var(--muted-text, #84969a);
  font-size: 12px;
}

.map-legend span {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.legend-dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #9fb5bc;
}

.legend-dot.extraction {
  background: #8bd0be;
}

.node-inspector {
  position: sticky;
  top: 18px;
}

.extraction-detail {
  border-left-color: #71bea9;
}

@media (max-width: 980px) {
  .map-layout {
    grid-template-columns: 1fr;
  }

  .node-inspector {
    position: static;
  }
}
</style>
