<!-- 日志页：浏览历史挂机会话、单局摘要与可解释行动报告。 -->
<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Select, Warning } from '@element-plus/icons-vue'
import api, { getApiError } from '../api'
import SessionTimeline from '../components/SessionTimeline.vue'
import type { ActionStyle, GameMap, LootSummary, Session, SessionDetail, SessionEvent, SessionRun, Weapon } from '../types'

const props = defineProps<{
  sessions: Session[]
  maps: GameMap[]
  weapons: Weapon[]
}>()

const emit = defineEmits<{ refresh: [] }>()
const selectedId = ref<number | null>(null)
const detail = ref<SessionDetail | null>(null)
const events = ref<SessionEvent[]>([])
const loading = ref(false)
const aborting = ref(false)

watchEffect(() => {
  if (!props.sessions.some((item) => item.id === selectedId.value)) {
    selectedId.value = props.sessions[0]?.id ?? null
  }
  if (selectedId.value && detail.value?.session.id !== selectedId.value) void loadSession(selectedId.value)
})

const runs = computed<SessionRun[]>(() => (detail.value?.runs ?? []).map((run) => ({
  ...run,
  loot: parseLoot(run.loot),
  report: parseReport(run.report),
})))

function parseLoot(raw: string): LootSummary[] {
  try {
    const value: unknown = JSON.parse(raw)
    return Array.isArray(value) && value.every((item) => isLootSummary(item)) ? value : []
  } catch {
    return []
  }
}

function isLootSummary(value: unknown): value is LootSummary {
  if (!value || typeof value !== 'object') return false
  const item = value as Record<string, unknown>
  return typeof item.id === 'string' && typeof item.itemId === 'string' && typeof item.name === 'string' && typeof item.category === 'string'
    && typeof item.quantity === 'number' && typeof item.containerId === 'string' && typeof item.source === 'string'
}

function parseReport(raw: string): string[] {
  try {
    const value: unknown = JSON.parse(raw)
    return Array.isArray(value) && value.every((line) => typeof line === 'string') ? value : ['报告格式异常']
  } catch {
    return ['报告读取失败']
  }
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function nameOf(id: string, list: Array<{ id: string; name: string }>) {
  return list.find((item) => item.id === id)?.name || id
}

function styleLabel(style: ActionStyle | string) {
  return ({ balanced: '均衡型', stealth: '隐秘型', aggressive: '激进型', greedy: '贪婪型' } as Record<string, string>)[style] || style || '均衡型'
}

async function loadSession(id: number) {
  selectedId.value = id
  loading.value = true
  events.value = []
  try {
    const [detailResponse, eventResponse] = await Promise.all([
      api.get<SessionDetail>(`/session/${id}`),
      api.get<SessionEvent[]>(`/session/${id}/events`),
    ])
    detail.value = detailResponse.data
    events.value = eventResponse.data
  } catch (error) {
    ElMessage.error(getApiError(error, '行动日志读取失败'))
  } finally {
    loading.value = false
  }
}

async function abortSession() {
  if (!selectedId.value) return
  aborting.value = true
  try {
    await api.post(`/session/${selectedId.value}/abort`)
    ElMessage.success('已提交中止信号')
    await loadSession(selectedId.value)
    emit('refresh')
  } catch (error) {
    ElMessage.error(getApiError(error, '中止行动失败'))
  } finally {
    aborting.value = false
  }
}
</script>

<template>
  <section class="view-page logs-view">
    <header class="page-heading"><div><span class="eyebrow">行动记录</span><h1>日志</h1><p>复盘关键判定、战斗代价和每次撤离结果。</p></div></header>
    <div v-if="sessions.length" class="logs-layout">
      <aside class="session-list">
        <button v-for="session in sessions" :key="session.id" type="button" :class="{ active: session.id === selectedId }" @click="loadSession(session.id)">
          <span>#{{ session.id.toString().padStart(4, '0') }}</span>
          <strong>{{ styleLabel(session.style) }}</strong>
          <small>{{ formatTime(session.startTime) }} · {{ session.totalRuns }} 局</small>
          <i :class="session.status">{{ session.status === 'finished' ? '已完成' : session.status === 'aborted' ? '已中止' : session.status === 'failed' ? '执行失败' : session.status === 'waiting_injury' ? '伤势恢复中' : '进行中' }}</i>
        </button>
      </aside>

      <section v-loading="loading" class="log-detail surface-panel">
        <template v-if="detail">
          <div class="log-summary">
            <div><span>行动编号</span><strong>#{{ detail.session.id.toString().padStart(4, '0') }}</strong></div>
            <div><span>区域</span><strong>{{ nameOf(detail.session.mapId, maps) }}</strong></div>
            <div><span>风格 / 失能预案</span><strong>{{ styleLabel(detail.session.style) }} · 预设 {{ detail.session.recoveryPreset }}</strong></div>
            <div><span>武器</span><strong>{{ nameOf(detail.session.weaponId, weapons) }}</strong></div>
            <div><span>随机种子</span><strong>{{ String(detail.session.seed).slice(-8) }}</strong></div>
            <el-button v-if="detail.session.status === 'running' || detail.session.status === 'waiting_injury'" type="danger" plain :loading="aborting" @click="abortSession">中止行动</el-button>
          </div>
          <section class="history-timeline">
            <div class="panel-heading"><div><span>01</span><h2>事件时间线</h2></div><small>{{ events.length }} 条已记录事件</small></div>
            <SessionTimeline :events="events" compact />
          </section>
          <div class="run-list">
            <details v-for="run in runs" :key="run.id" class="run-entry" :open="run.runIndex === runs.length">
              <summary>
                <span class="result-icon" :class="run.result"><el-icon><Select v-if="run.result === 'success'" /><Warning v-else /></el-icon></span>
                <div><strong>第 {{ run.runIndex }} 局 · {{ run.result }}</strong><small>{{ run.durationMin }} 分钟 · 热度 {{ run.heat }} · 弹药 {{ run.ammoUsed }}</small></div>
                <b>{{ run.injury === 'none' ? '无伤' : run.injury }}</b>
              </summary>
              <div v-if="run.loot.length" class="run-loot"><span v-for="item in run.loot" :key="item.id">{{ item.name }} x{{ item.quantity }}</span></div>
              <pre>{{ run.report.join('\n') }}</pre>
            </details>
          </div>
        </template>
      </section>
    </div>
    <div v-else class="page-empty"><el-icon><Document /></el-icon><strong>暂无行动日志</strong><span>完成首次探索后，报告会保存在这里。</span></div>
  </section>
</template>
