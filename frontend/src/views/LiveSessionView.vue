<!-- 实时行动页：通过 SSE 播放在线事件，断线后从数据库事件游标继续。 -->
<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { ElMessage } from 'element-plus'
import { DataLine, MapLocation, Timer, VideoPlay, Warning } from '@element-plus/icons-vue'
import api, { getApiError } from '../api'
import SessionTimeline from '../components/SessionTimeline.vue'
import { useSessionStream } from '../composables/useSessionStream'
import type { GameMap, MapNode, Player, Session, SessionEvent } from '../types'

const props = defineProps<{
  sessionId: number
  player: Player
  maps: GameMap[]
  nodes: MapNode[]
}>()

const emit = defineEmits<{
  refresh: []
  openLogs: []
}>()

const aborting = ref(false)
const { session, events, loading, errorMessage, now, loadSession, reconnectNow } = useSessionStream(toRef(props, 'sessionId'), () => emit('refresh'))

const currentNodeEvent = computed(() => [...events.value].reverse().find((event) => event.eventType === 'node_entered') ?? null)
const latestBattleStart = computed(() => [...events.value].reverse().find((event) => event.eventType === 'battle_started') ?? null)
const latestBattleFinished = computed(() => {
  if (!latestBattleStart.value) return null
  return [...events.value].reverse().find((event) => event.eventType === 'battle_finished' && event.id > latestBattleStart.value!.id) ?? null
})
const activeBattle = computed(() => {
  if (!latestBattleStart.value || latestBattleFinished.value) return null
  return latestBattleStart.value
})
const battleSpotlight = computed(() => latestBattleStart.value)
const latestBattleMetrics = computed(() => {
  if (!battleSpotlight.value) return null
  return [...events.value].reverse().find((event) => event.eventType.startsWith('battle_') && event.id >= battleSpotlight.value!.id) ?? null
})
const progressPercent = computed(() => {
  if (!session.value?.currentRunStartedAt || !session.value.nextRunAt) return session.value?.status === 'finished' ? 100 : 0
  const start = Date.parse(session.value.currentRunStartedAt)
  const end = Date.parse(session.value.nextRunAt)
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return 0
  return Math.min(100, Math.max(0, ((now.value - start) / (end - start)) * 100))
})
const liveElapsedSec = computed(() => {
  if (!session.value) return 0
  const base = session.value.elapsedSec
  if (!session.value.currentRunStartedAt) return base
  const start = Date.parse(session.value.currentRunStartedAt)
  const end = session.value.nextRunAt ? Date.parse(session.value.nextRunAt) : now.value
  if (!Number.isFinite(start) || !Number.isFinite(end)) return base
  return base + Math.max(0, Math.min(Math.floor((now.value - start) / 1000), Math.floor((end - start) / 1000)))
})

function payloadNumber(event: SessionEvent | null, key: string): number {
  const value = event?.payload[key]
  return typeof value === 'number' ? value : 0
}

function payloadText(event: SessionEvent | null, key: string, fallback: string): string {
  const value = event?.payload[key]
  return typeof value === 'string' ? value : fallback
}

function metricText(key: string): string {
  const event = [...events.value].reverse().find((item) => typeof item.payload[key] === 'number')
  const value = event?.payload[key]
  return typeof value === 'number' ? Math.round(value).toString() : '—'
}

function healthPercent(event: SessionEvent | null, currentKey: string, maxKey: string): number {
  const maximum = payloadNumber(event, maxKey)
  if (maximum <= 0) return 0
  return Math.min(100, Math.max(0, (payloadNumber(event, currentKey) / maximum) * 100))
}

function statusLabel(status: Session['status']): string {
  return ({ running: '行动中', waiting_injury: '伤势恢复中', finished: '已完成', aborted: '已中止', failed: '执行失败' } as Record<string, string>)[status] || status
}

function formatDuration(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  return `${minutes}分 ${remainder.toString().padStart(2, '0')}秒`
}

function mapName(mapID: string): string {
  return props.maps.find((map) => map.id === mapID)?.name || mapID
}

function currentNodeName(): string {
  const name = payloadText(currentNodeEvent.value, 'name', '')
  if (name) return name
  return props.nodes.find((node) => node.id === currentNodeEvent.value?.nodeId)?.name || '等待第一个节点'
}

async function abortSession() {
  aborting.value = true
  try {
    await api.post(`/session/${props.sessionId}/abort`)
    ElMessage.success('已提交中止信号')
    await loadSession()
    emit('refresh')
  } catch (error) {
    ElMessage.error(getApiError(error, '中止行动失败'))
  } finally {
    aborting.value = false
  }
}

</script>

<template>
  <section class="view-page live-session-view">
    <header class="page-heading live-heading">
      <div>
        <span class="eyebrow">实时行动</span>
        <h1>{{ session ? `行动 #${session.id.toString().padStart(4, '0')}` : '实时行动' }}</h1>
        <p>{{ session ? `${mapName(session.mapId)} · ${statusLabel(session.status)}` : '正在连接行动事件流' }}</p>
      </div>
      <div v-if="session" class="live-status-badge">
        <span class="status-dot" :class="{ warning: session.status === 'waiting_injury' }" />
        <strong>{{ statusLabel(session.status) }}</strong>
        <small>{{ events.length }} 条事件</small>
      </div>
    </header>

    <div v-if="loading" class="loading-shell"><el-skeleton :rows="8" animated /></div>
    <div v-else-if="errorMessage" class="fatal-state"><strong>实时行动无法载入</strong><p>{{ errorMessage }}</p><el-button @click="reconnectNow">重新连接</el-button></div>
    <template v-else-if="session">
      <section class="live-progress surface-panel">
        <div class="live-progress__top">
          <div><span>当前节点</span><strong>{{ currentNodeName() }}</strong></div>
          <div><span>游戏时间</span><strong>{{ formatDuration(liveElapsedSec) }}</strong></div>
          <div><span>局次</span><strong>{{ session.totalRuns + (session.status === 'running' ? 1 : 0) }}</strong></div>
          <el-button v-if="session.status === 'running' || session.status === 'waiting_injury'" type="danger" plain :loading="aborting" @click="abortSession">中止行动</el-button>
          <el-button v-else @click="emit('openLogs')">查看历史</el-button>
        </div>
        <el-progress :percentage="progressPercent" :show-text="false" />
        <div class="live-progress__foot"><span>{{ session.currentRunStartedAt ? '当前局推进中，事件会按时间轴逐条出现' : '等待下一局调度' }}</span><b>{{ Math.round(progressPercent) }}%</b></div>
      </section>

      <div class="live-layout">
        <section class="live-feed surface-panel">
          <div class="panel-heading"><div><span>01</span><h2>探索过程</h2></div><small>SSE 实时事件流</small></div>
          <div v-if="battleSpotlight" class="battle-spotlight">
            <div class="battle-spotlight__heading"><span><el-icon><VideoPlay /></el-icon>{{ activeBattle ? '正在交战' : '战斗回放' }}</span><strong>{{ payloadText(battleSpotlight, 'target', battleSpotlight.subjectId) }}</strong></div>
            <div class="battle-bars">
              <div><span>行动员</span><el-progress :percentage="healthPercent(latestBattleMetrics || battleSpotlight, 'playerHp', 'playerMaxHp')" :show-text="false" /><b>{{ Math.round(payloadNumber(latestBattleMetrics || battleSpotlight, 'playerHp')) }} HP</b></div>
              <div><span>敌方</span><el-progress status="exception" :percentage="healthPercent(latestBattleMetrics || battleSpotlight, 'enemyHp', 'enemyMaxHp')" :show-text="false" /><b>{{ Math.round(payloadNumber(latestBattleMetrics || battleSpotlight, 'enemyHp')) }} HP</b></div>
            </div>
          </div>
          <SessionTimeline :events="events" />
        </section>

        <aside class="live-side">
          <section class="live-map surface-panel">
            <div class="panel-heading"><div><span>02</span><h2><el-icon><MapLocation /></el-icon>路线</h2></div><small>{{ mapName(session.mapId) }}</small></div>
            <div class="live-map__image"><img src="/city-map.svg" alt="当前行动区域地图" /><div class="live-map__node"><el-icon><MapLocation /></el-icon><strong>{{ currentNodeName() }}</strong></div></div>
          </section>

          <section class="live-metrics surface-panel">
            <div class="panel-heading"><div><span>03</span><h2><el-icon><DataLine /></el-icon>行动数据</h2></div><small>当前可见状态</small></div>
            <div class="metric-grid">
              <div><span>生命</span><strong>{{ metricText('playerHp') }}</strong></div>
              <div><span>压力</span><strong>{{ metricText('playerStress') }}</strong></div>
              <div><span>热度</span><strong>{{ metricText('heat') }}</strong></div>
              <div><span>弹药</span><strong>{{ metricText('playerAmmo') }}</strong></div>
              <div><span>护甲</span><strong>{{ metricText('playerArmorDurability') }}</strong></div>
            </div>
            <div class="live-operator"><el-icon><Timer /></el-icon><div><span>行动员</span><strong>{{ player.name }}</strong></div><i class="status-dot" /></div>
          </section>

          <section v-if="session.status === 'waiting_injury'" class="live-notice surface-panel"><el-icon><Warning /></el-icon><div><strong>伤势恢复中</strong><span>恢复完成后会自动安排下一局</span></div></section>
        </aside>
      </div>
    </template>
  </section>
</template>
