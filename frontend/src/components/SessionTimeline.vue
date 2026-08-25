<!-- Session 事件时间线：实时行动与离线历史共用的结构化事件展示。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { Box, Compass, Document, Select, VideoPlay, Warning } from '@element-plus/icons-vue'
import type { SessionEvent } from '../types'

const props = defineProps<{
  events: SessionEvent[]
  compact?: boolean
}>()

const events = computed(() => [...props.events].sort((left, right) => left.id - right.id))

function valueOf(event: SessionEvent, key: string): unknown {
  return event.payload[key]
}

function textOf(event: SessionEvent, key: string, fallback = ''): string {
  const value = valueOf(event, key)
  return typeof value === 'string' ? value : fallback
}

function numberOf(event: SessionEvent, key: string, fallback = 0): number {
  const value = valueOf(event, key)
  return typeof value === 'number' ? value : fallback
}

function formatTime(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(value))
}

function eventTitle(event: SessionEvent): string {
  switch (event.eventType) {
    case 'run_started': return `第 ${event.runIndex} 局开始`
    case 'node_entered': return `抵达 ${textOf(event, 'name', event.nodeId)}`
    case 'event_triggered': return textOf(event, 'name', '触发事件')
    case 'evacuation_started': return '行动转入撤离'
    case 'container_search_started': return `发现容器：${textOf(event, 'name', event.subjectId)}`
    case 'loot_found': return `发现物资：${textOf(event, 'name', event.subjectId)}`
    case 'loot_collected': return `物资已装入携行：${textOf(event, 'name', event.subjectId)}`
    case 'container_search_finished': return '容器搜索完成'
    case 'battle_started': return `遭遇敌人：${textOf(event, 'target', event.subjectId)}`
    case 'battle_attack': return `${textOf(event, 'actor', '攻击方')} → ${textOf(event, 'target', '目标')}`
    case 'battle_round': return `战斗第 ${numberOf(event, 'round')} 轮`
    case 'battle_escape': return '脱离判定'
    case 'battle_finished': return '战斗结束'
    case 'loot_extracted': return `成功带出：${textOf(event, 'name', event.subjectId)}`
    case 'loot_stored': return `已存入仓库：${textOf(event, 'name', event.subjectId)}`
    case 'loot_overflow': return `仓库不足，放弃：${textOf(event, 'name', event.subjectId)}`
    case 'ammo_refilled': return textOf(event, 'source') === 'preset_warehouse' ? '预设弹药已装入' : '弹药自动补给'
    case 'run_settled': return `第 ${event.runIndex} 局已结算`
    case 'session_finished': return '行动完成'
    case 'session_aborted': return '行动已中止'
    case 'session_failed': return '行动执行失败'
    default: return event.eventType
  }
}

function eventSummary(event: SessionEvent): string {
  switch (event.eventType) {
    case 'run_started': return textOf(event, 'mapName', '开始推进路线')
    case 'node_entered': return `${textOf(event, 'distance', '未知距离')} · 探索 ${numberOf(event, 'exploreTime')} 分钟`
    case 'event_triggered': return `${valueOf(event, 'success') === false ? '判定未通过' : textOf(event, 'intent', '自动决策')} · ${textOf(event, 'phase')}`
    case 'evacuation_started': return `${textOf(event, 'reason', '未知原因')}${valueOf(event, 'emergency') === true ? ' · 紧急' : ''}`
    case 'container_search_started': return `${textOf(event, 'source', '节点搜索')} · 搜索 ${numberOf(event, 'searchTime')} 分钟`
    case 'loot_found': return `${numberOf(event, 'quantity')} 件 · ${valueOf(event, 'collected') === false ? '未能装入携行' : '等待继续推进'}`
    case 'loot_collected': return `${numberOf(event, 'quantity')} 件已计入本局携行`
    case 'container_search_finished': return `本次搜索获得 ${numberOf(event, 'quantity')} 件可携带物资`
    case 'battle_started': return '自动战斗播放中'
    case 'battle_attack': {
      if (valueOf(event, 'hit') !== true) return `未命中 · 命中率 ${numberOf(event, 'hitRate').toFixed(1)}% · 掷 ${numberOf(event, 'hitRoll')}`
      const healthDamage = numberOf(event, 'healthDamage').toFixed(1)
      if (textOf(event, 'hitLocation') === 'limb') return `命中无甲四肢 · 肉伤 ${numberOf(event, 'fleshDamage').toFixed(1)} · HP -${healthDamage}`
      const penetrated = valueOf(event, 'penetrated') === true ? '穿透' : '未穿透'
      return `命中护甲 · N${numberOf(event, 'ammoLevel')} 对 A${numberOf(event, 'armorLevel')}（有效 A${numberOf(event, 'effectiveArmorLevel')}）· ${penetrated} · HP -${healthDamage} · 护甲 -${numberOf(event, 'armorDamage').toFixed(1)}`
    }
    case 'battle_round': return `玩家 ${numberOf(event, 'playerHp').toFixed(1)} HP / 护甲 ${numberOf(event, 'playerArmorDurability').toFixed(1)} · 敌人 ${numberOf(event, 'enemyHp').toFixed(1)} HP / 护甲 ${numberOf(event, 'enemyArmorDurability').toFixed(1)} · 弹药 ${numberOf(event, 'playerAmmo')}`
    case 'battle_escape': return `${textOf(event, 'message', '脱离判定完成')} · ${valueOf(event, 'success') === true ? '成功' : '失败'}`
    case 'battle_finished': return `结果：${textOf(event, 'winner', '未知')}`
    case 'loot_extracted': return `${numberOf(event, 'quantity')} 件成功撤离地图`
    case 'loot_stored': return `${numberOf(event, 'quantity')} 件已写入基地仓库`
    case 'loot_overflow': return `${numberOf(event, 'quantity')} 件因仓库容量不足被放弃`
    case 'ammo_refilled': {
      const rounds = numberOf(event, 'rounds')
      const toLevel = numberOf(event, 'toLevel')
      if (textOf(event, 'source') === 'preset_warehouse') return `从仓库装入 ${rounds} 发 N${toLevel} 预设弹药`
      return `N${numberOf(event, 'fromLevel')} → N${toLevel} · ${rounds} 发 · 花费 ￥${numberOf(event, 'totalPrice').toLocaleString()}`
    }
    case 'run_settled': return `结果：${textOf(event, 'result', '未知')} · 热度 ${numberOf(event, 'heat')} · 弹药 ${numberOf(event, 'ammoUsed')}`
    case 'session_finished': return '后续调度已停止，结果已保存'
    case 'session_aborted': return '用户中止了当前行动，未结算的未来事件不会继续显示'
    case 'session_failed': return '后台执行失败，请查看服务日志'
    default: return ''
  }
}

function eventClass(event: SessionEvent): string {
  if (event.eventType.startsWith('battle_')) return 'battle'
  if (event.eventType.startsWith('loot_') || event.eventType.startsWith('container_') || event.eventType === 'ammo_refilled') return 'loot'
  if (event.eventType === 'session_finished' || event.eventType === 'run_settled') return 'success'
  if (event.eventType === 'session_aborted' || event.eventType === 'session_failed') return 'danger'
  return 'normal'
}
</script>

<template>
  <div class="session-timeline" :class="{ compact }">
    <div v-if="!events.length" class="timeline-empty">
      <el-icon><Document /></el-icon>
      <span>还没有可显示的探索事件</span>
    </div>
    <ol v-else class="timeline-list">
      <li v-for="event in events" :key="event.id" class="timeline-item" :class="eventClass(event)">
        <div class="timeline-marker">
          <el-icon v-if="event.eventType.startsWith('container_') || event.eventType.startsWith('loot_') || event.eventType === 'ammo_refilled'"><Box /></el-icon>
          <el-icon v-else-if="event.eventType.startsWith('battle_')"><VideoPlay /></el-icon>
          <el-icon v-else-if="event.eventType === 'node_entered'"><Compass /></el-icon>
          <el-icon v-else-if="eventClass(event) === 'success'"><Select /></el-icon>
          <el-icon v-else-if="eventClass(event) === 'danger'"><Warning /></el-icon>
          <el-icon v-else><Document /></el-icon>
        </div>
        <div class="timeline-body">
          <div class="timeline-meta"><span>第 {{ event.runIndex }} 局 · {{ formatTime(event.availableAt) }}</span><b>#{{ event.id }}</b></div>
          <strong>{{ eventTitle(event) }}</strong>
          <p v-if="eventSummary(event)">{{ eventSummary(event) }}</p>

          <div v-if="event.eventType === 'container_search_started'" class="timeline-artwork container-artwork">
            <el-icon><Box /></el-icon>
            <span>容器搜索</span>
          </div>
          <div v-if="event.eventType === 'battle_started'" class="timeline-artwork battle-artwork">
            <el-icon><VideoPlay /></el-icon>
            <span>自动战斗</span>
          </div>
          <div v-if="event.eventType === 'loot_found' || event.eventType === 'loot_collected' || event.eventType.startsWith('loot_')" class="timeline-loot">
            <span>{{ textOf(event, 'name', event.subjectId) }}</span>
            <b>x{{ numberOf(event, 'quantity') }}</b>
          </div>
        </div>
      </li>
    </ol>
  </div>
</template>
