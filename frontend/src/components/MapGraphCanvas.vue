<!-- 地图图画布：统一渲染节点、移动边、撤离点、已访问状态和规划路线。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { Location } from '@element-plus/icons-vue'
import type { MapGraph, MapNode } from '../types'

const props = withDefaults(defineProps<{
  graph: MapGraph
  selectedNodeId?: string
  currentNodeId?: string
  visitedNodeIds?: string[]
  routeNodeIds?: string[]
  compact?: boolean
}>(), {
  selectedNodeId: '',
  currentNodeId: '',
  visitedNodeIds: () => [],
  routeNodeIds: () => [],
  compact: false,
})

const emit = defineEmits<{
  select: [nodeId: string]
}>()

const nodeByID = computed(() => new Map(props.graph.nodes.map((node) => [node.id, node])))
const visited = computed(() => new Set(props.visitedNodeIds))
const route = computed(() => new Set(props.routeNodeIds))
const routeSegments = computed(() => {
  const segments = new Set<string>()
  for (let index = 1; index < props.routeNodeIds.length; index += 1) {
    segments.add(routeSegmentKey(props.routeNodeIds[index - 1], props.routeNodeIds[index]))
  }
  return segments
})

function nodePosition(node: MapNode): { left: string; top: string } {
  const columns = Math.max(props.graph.map.layoutColumns, 1)
  const rows = Math.max(props.graph.map.layoutRows, 1)
  return {
    left: `${((node.positionX + 0.5) / columns) * 100}%`,
    top: `${((node.positionY + 0.5) / rows) * 100}%`,
  }
}

function nodeNumber(node: MapNode): string {
  const match = node.id.match(/(?:^|_)node_(\d+)$/)
  return match?.[1]?.padStart(2, '0') || node.id.slice(-2)
}

function nodeAt(id: string): MapNode | undefined {
  return nodeByID.value.get(id)
}

function routeSegmentKey(fromNodeId: string, toNodeId: string): string {
  return `${fromNodeId}\u0000${toNodeId}`
}

function isRouteEdgeHighlighted(edge: MapGraph['edges'][number]): boolean {
  if (routeSegments.value.has(routeSegmentKey(edge.fromNodeId, edge.toNodeId))) return true
  return edge.bidirectional && routeSegments.value.has(routeSegmentKey(edge.toNodeId, edge.fromNodeId))
}

function pointPosition(anchorNodeId: string): { left: string; top: string } {
  const node = nodeAt(anchorNodeId)
  return node ? nodePosition(node) : { left: '50%', top: '50%' }
}
</script>

<template>
  <div
    class="map-graph-canvas"
    :class="{ compact }"
    :style="{ '--graph-columns': Math.max(graph.map.layoutColumns, 1), '--graph-rows': Math.max(graph.map.layoutRows, 1) }"
    role="img"
    :aria-label="`${graph.map.name} 节点拓扑图`"
  >
    <svg
      class="map-graph-canvas__edges"
      :viewBox="`0 0 ${Math.max(graph.map.layoutColumns, 1)} ${Math.max(graph.map.layoutRows, 1)}`"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <line
        v-for="edge in graph.edges"
        :key="edge.id"
        :class="{ highlighted: isRouteEdgeHighlighted(edge) }"
        :x1="(nodeAt(edge.fromNodeId)?.positionX ?? 0) + 0.5"
        :y1="(nodeAt(edge.fromNodeId)?.positionY ?? 0) + 0.5"
        :x2="(nodeAt(edge.toNodeId)?.positionX ?? 0) + 0.5"
        :y2="(nodeAt(edge.toNodeId)?.positionY ?? 0) + 0.5"
      />
    </svg>

    <div
      v-for="point in graph.extractionPoints.filter((item) => item.enabled)"
      :key="point.id"
      class="map-graph-canvas__extraction"
      :style="pointPosition(point.anchorNodeId)"
      :title="`${point.name} · ${point.travelTime} 分钟`"
    >
      <span class="map-graph-canvas__extraction-icon"><el-icon><Location /></el-icon></span>
      <small>{{ point.name }}</small>
    </div>

    <button
      v-for="node in graph.nodes"
      :key="node.id"
      type="button"
      class="map-graph-canvas__node"
      :class="{
        selected: node.id === selectedNodeId,
        current: node.id === currentNodeId,
        visited: visited.has(node.id),
        planned: route.has(node.id),
        anchor: graph.extractionPoints.some((point) => point.enabled && point.anchorNodeId === node.id),
      }"
      :style="nodePosition(node)"
      :aria-label="`${node.name}，节点${nodeNumber(node)}`"
      @click="emit('select', node.id)"
    >
      <span>{{ nodeNumber(node) }}</span>
      <strong>{{ node.name }}</strong>
    </button>
  </div>
</template>

<style scoped>
.map-graph-canvas {
  position: relative;
  min-height: 520px;
  overflow: hidden;
  border: 1px solid rgba(118, 142, 156, 0.26);
  background:
    linear-gradient(rgba(22, 34, 39, 0.76), rgba(22, 34, 39, 0.76)),
    url('/city-map.svg') center / cover;
  isolation: isolate;
}

.map-graph-canvas::before {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(rgba(168, 192, 197, 0.08) 1px, transparent 1px), linear-gradient(90deg, rgba(168, 192, 197, 0.08) 1px, transparent 1px);
  background-size: calc(100% / var(--graph-columns)) calc(100% / var(--graph-rows));
  content: '';
  pointer-events: none;
}

.map-graph-canvas__edges {
  position: absolute;
  z-index: 1;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.map-graph-canvas__edges line {
  stroke: rgba(159, 181, 188, 0.46);
  stroke-width: 0.035;
  vector-effect: non-scaling-stroke;
}

.map-graph-canvas__edges line.highlighted {
  stroke: #d9a441;
  stroke-width: 0.075;
}

.map-graph-canvas__node,
.map-graph-canvas__extraction {
  position: absolute;
  transform: translate(-50%, -50%);
}

.map-graph-canvas__node {
  z-index: 2;
  display: grid;
  width: 122px;
  min-height: 68px;
  padding: 8px 9px;
  border: 1px solid rgba(189, 211, 215, 0.55);
  border-radius: 6px;
  color: #e8f0ee;
  background: rgba(22, 34, 39, 0.9);
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.2);
  cursor: pointer;
  text-align: left;
  transition: border-color 160ms ease, background 160ms ease, transform 160ms ease;
}

.map-graph-canvas__node:hover,
.map-graph-canvas__node.selected {
  border-color: #e0b45b;
  background: rgba(43, 59, 62, 0.98);
  transform: translate(-50%, -50%) translateY(-3px);
}

.map-graph-canvas__node.planned {
  border-color: rgba(217, 164, 65, 0.78);
}

.map-graph-canvas__node.current {
  box-shadow: 0 0 0 3px rgba(221, 91, 66, 0.32), 0 8px 22px rgba(0, 0, 0, 0.25);
}

.map-graph-canvas__node.visited {
  background: rgba(43, 67, 63, 0.94);
}

.map-graph-canvas__node span {
  color: #d9a441;
  font: 700 11px/1.2 ui-monospace, SFMono-Regular, Consolas, monospace;
  letter-spacing: 0.08em;
}

.map-graph-canvas__node strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 650;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.map-graph-canvas__extraction {
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: -46px;
  padding: 4px 7px;
  border: 1px solid rgba(106, 190, 172, 0.7);
  border-radius: 999px;
  color: #bce8dc;
  background: rgba(19, 65, 58, 0.94);
  white-space: nowrap;
}

.map-graph-canvas__extraction-icon {
  display: grid;
  width: 18px;
  height: 18px;
  place-items: center;
  border-radius: 50%;
  color: #133c35;
  background: #8bd0be;
  font-size: 12px;
  font-weight: 800;
}

.map-graph-canvas__extraction small {
  font-size: 11px;
  font-weight: 700;
}

.map-graph-canvas.compact {
  min-height: 360px;
}

.map-graph-canvas.compact .map-graph-canvas__node {
  width: 94px;
  min-height: 54px;
  padding: 6px;
}

.map-graph-canvas.compact .map-graph-canvas__node strong {
  font-size: 11px;
}

.map-graph-canvas.compact .map-graph-canvas__extraction {
  margin-top: -38px;
}

@media (max-width: 720px) {
  .map-graph-canvas {
    min-height: 430px;
  }

  .map-graph-canvas__node {
    width: 94px;
    min-height: 58px;
  }

  .map-graph-canvas__node strong {
    font-size: 11px;
  }

  .map-graph-canvas__extraction small {
    display: none;
  }
}
</style>
