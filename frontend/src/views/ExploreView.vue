<!-- 探索部署页：配置地图、行动风格与失能后的预设装备，并执行风险预测与挂机。 -->
<script setup lang="ts">
import { computed, ref, watch, watchEffect } from 'vue'
import { ElMessage } from 'element-plus'
import { Compass, Refresh, VideoPlay, Warning } from '@element-plus/icons-vue'
import api, { getApiError } from '../api'
import { presetOf } from '../types'
import type {
  Armor,
  ActionStyle,
  Consumable,
  GameMap,
  Player,
  PlayerLoadout,
  PreviewResult,
  Session,
  StartSessionRequest,
  Weapon,
} from '../types'

const props = defineProps<{
  player: Player
  loadout: PlayerLoadout
  maps: GameMap[]
  weapons: Weapon[]
  armors: Armor[]
  consumables: Consumable[]
}>()

const emit = defineEmits<{
  created: [session: Session]
}>()

const styles = [
  { value: 'balanced' as ActionStyle, label: '均衡型', desc: '在收益与风险之间保持平衡，普通巡逻优先绕行' },
  { value: 'stealth' as ActionStyle, label: '隐秘型', desc: '优先避战、低热度和安全容器，较早返回撤离路线' },
  { value: 'aggressive' as ActionStyle, label: '激进型', desc: '主动伏击和清除敌人，接受更高战斗与热度代价' },
  { value: 'greedy' as ActionStyle, label: '贪婪型', desc: '优先高价值物资与情报，愿意为收益延后撤离' },
]

const selectedMap = ref('')
const selectedStyle = ref<ActionStyle>('balanced')
const selectedPreset = ref(1)
const preview = ref<PreviewResult | null>(null)
const previewing = ref(false)
const starting = ref(false)

watchEffect(() => {
  if (!props.maps.some((item) => item.id === selectedMap.value)) selectedMap.value = props.maps[0]?.id ?? ''
})

watch([selectedMap, selectedStyle, selectedPreset], () => {
  preview.value = null
})

function presetSummary(index: number) {
  const slot = presetOf(props.loadout, index)
  const weapon = props.weapons.find((item) => item.id === slot.weaponId)
  const armor = props.armors.find((item) => item.id === slot.armorId)
  if (!slot.weaponId || !slot.armorId || !weapon || !armor) return '未配置，请先在角色页面设置'
  const supplies = slot.consumables.map((id) => props.consumables.find((item) => item.id === id)?.name ?? id).join('、')
  return `${weapon.name} · ${armor.name}${supplies ? ` · ${supplies}` : ' · 无补给'}`
}

const selectedPresetSummary = computed(() => presetSummary(selectedPreset.value))
const canSubmit = computed(() => Boolean(
  selectedMap.value && props.loadout.weaponId && props.loadout.armorId
  && presetOf(props.loadout, selectedPreset.value).weaponId && presetOf(props.loadout, selectedPreset.value).armorId,
))

function buildRequest(): StartSessionRequest {
  return {
    mapId: selectedMap.value,
    style: selectedStyle.value,
    recoveryPreset: selectedPreset.value,
  }
}

async function loadPreview() {
  previewing.value = true
  try {
    const { data } = await api.post<PreviewResult>('/session/preview', buildRequest())
    preview.value = data
  } catch (error) {
    ElMessage.error(getApiError(error, '风险预测失败'))
  } finally {
    previewing.value = false
  }
}

async function startSession() {
  starting.value = true
  try {
    const { data } = await api.post<Session>('/session/start', buildRequest())
    ElMessage.success(`行动 #${data.id} 已开始后台执行`)
    emit('created', data)
  } catch (error) {
    ElMessage.error(getApiError(error, '行动启动失败'))
  } finally {
    starting.value = false
  }
}
</script>

<template>
  <section class="view-page explore-view">
    <header class="page-heading">
      <div>
        <span class="eyebrow">行动部署</span>
        <h1>探索</h1>
        <p>确认区域与行动策略，系统将连续完成搜索、交战和撤离。</p>
      </div>
      <div class="operator-badge">
        <span class="operator-badge__icon"><el-icon><Compass /></el-icon></span>
        <div><span>当前行动员</span><strong>{{ player.name }}</strong></div>
        <i :class="player.injury && player.injury !== 'none' ? 'danger' : ''" />
      </div>
    </header>

    <div class="deployment-layout">
      <div class="deployment-form surface-panel">
        <div class="panel-heading">
          <div><span>01</span><h2>行动参数</h2></div>
          <small>配置在本次会话中锁定</small>
        </div>

        <div class="form-grid">
          <label class="field-group">
            <span>目标区域</span>
            <el-select v-model="selectedMap" size="large">
              <el-option v-for="item in maps" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </label>
          <label class="field-group">
            <span>行动风格</span>
            <el-select v-model="selectedStyle" size="large">
              <el-option v-for="item in styles" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </label>
        </div>

        <p class="style-note">{{ styles.find((item) => item.value === selectedStyle)?.desc }}</p>

        <div class="preset-block">
          <span class="preset-block__label">失能预案 · 丢装后按第 N 套预设继续探索</span>
          <div class="preset-picker">
            <button
              v-for="index in 3"
              :key="index"
              type="button"
              class="preset-card"
              :class="{ active: selectedPreset === index }"
              @click="selectedPreset = index"
            >
              <span class="preset-card__index">预设 {{ index }}</span>
              <span class="preset-card__summary">{{ presetSummary(index) }}</span>
            </button>
          </div>
          <small>当前所选：{{ selectedPresetSummary }}</small>
        </div>

        <div class="loadout-row">
          <div class="deployment-loadout">
            <span>当前携行</span>
            <strong>{{ weapons.find((item) => item.id === loadout.weaponId)?.name }} · {{ armors.find((item) => item.id === loadout.armorId)?.name }}</strong>
            <small>补给：{{ loadout.consumables.map((id) => consumables.find((item) => item.id === id)?.name ?? id).join('、') || '无' }}，装备配置请在角色页面调整</small>
          </div>
        </div>
      </div>

      <aside class="risk-panel surface-panel">
        <div class="panel-heading">
          <div><span>02</span><h2>风险预估</h2></div>
          <el-button :icon="Refresh" :loading="previewing" circle title="刷新预测" @click="loadPreview" />
        </div>

        <div v-if="preview" class="risk-metrics">
          <div v-for="(value, key) in preview" :key="key" class="risk-metric">
            <span>{{ key }}</span><strong>{{ value }}</strong>
          </div>
        </div>
        <div v-else class="risk-empty">
          <el-icon><Warning /></el-icon>
          <strong>尚未生成预测</strong>
          <span>基于当前配置执行 100 次快速模拟</span>
        </div>

        <div class="launch-block">
          <p><span class="status-dot" />{{ player.injury && player.injury !== 'none' ? '行动员暂不可出发' : '行动员状态正常' }}</p>
          <el-button
            type="primary"
            size="large"
            :icon="VideoPlay"
            :loading="starting"
            :disabled="!canSubmit || (player.injury !== '' && player.injury !== 'none')"
            @click="startSession"
          >开始探索</el-button>
        </div>
      </aside>
    </div>
  </section>
</template>
