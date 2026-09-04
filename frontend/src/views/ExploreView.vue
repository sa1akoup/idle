<!-- 探索部署页：配置地图、行动风格与失能后的预设装备，并启动后台行动。 -->
<script setup lang="ts">
import { Compass, VideoPlay } from '@element-plus/icons-vue'
import { useExploreDeployment, type ExploreEmit, type ExploreProps } from '../composables/useExploreDeployment'

const props = defineProps<ExploreProps>()
const emit = defineEmits<ExploreEmit>()
const {
  styles, selectedMap, selectedStyle, selectedPreset, starting,
  recoveryMethods, selectedHPRecoveryMethod, selectedEnergyRecoveryMethod, selectedHydrationRecoveryMethod,
  currentWeapon, currentArmor, hasCurrentWeapon, hasCurrentArmor, recoveryPending,
  presetLabel, recoveryMethodLabel, canSubmit, startSession,
} = useExploreDeployment(props, emit)

const visibleQuests = () => props.quests.filter((quest) => quest.status !== 'locked')

type ShoppingNeed = { key: string; label: string; detail: string; have: number; need: number }

function firHave(itemId: string): number {
  const stacked = props.inventory
    .filter((item) => item.itemId === itemId && item.raidExtract)
    .reduce((sum, item) => sum + item.quantity, 0)
  const instances = props.itemInstances.filter((item) => (
    item.itemId === itemId
    && item.raidExtract
    && item.locationType === 'inventory'
    && item.status === 'normal'
    && item.currentDurability > 0
  )).length
  return stacked + instances
}

function shoppingNeeds(): ShoppingNeed[] {
  const rows: ShoppingNeed[] = []
  for (const quest of props.quests) {
    if (quest.status !== 'active' || quest.objectiveType !== 'extract_item' || !quest.targetId) continue
    rows.push({
      key: `quest-${quest.id}`,
      label: `合同 · ${quest.name}`,
      detail: quest.targetLabel,
      have: firHave(quest.targetId),
      need: quest.required,
    })
  }
  for (const facility of props.hideout?.facilities ?? []) {
    const upgrade = facility.nextUpgrade
    if (!upgrade) continue
    for (const requirement of upgrade.requirements) {
      if (requirement.requirementType !== 'item' || !requirement.referenceId) continue
      rows.push({
        key: `hideout-${facility.id}-${requirement.referenceId}`,
        label: `${facility.name} LV.${upgrade.level}`,
        detail: `${requirement.label}（局内带出）`,
        have: firHave(requirement.referenceId),
        need: requirement.quantity,
      })
    }
  }
  return rows
}

// 随身弹药只读摘要：弹药槽在角色页配置，这里仅展示名称与发数。
function carriedAmmoSummary(): string {
  const cells = props.loadout.carriedAmmo ?? []
  const parts = cells
    .filter((cell) => cell.ammoId && cell.rounds > 0)
    .map((cell) => {
      const ammo = props.ammos?.find((item) => item.id === cell.ammoId)
      return `${ammo?.name ?? cell.ammoId} ×${cell.rounds}`
    })
  return parts.length ? parts.join('、') : '未配置'
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
        <i :class="player.hp <= 0 || player.energy <= 0 || player.hydration <= 0 ? 'danger' : ''" />
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

        <div class="recovery-block">
          <div class="recovery-block__heading">
            <span>行动后自动恢复</span>
            <small>撤离成功或失能后按此配置自动执行</small>
          </div>
          <div class="recovery-grid">
            <label class="field-group">
              <span>生命 · 恢复至 100%</span>
              <el-select v-model="selectedHPRecoveryMethod" size="large">
                <el-option v-for="item in recoveryMethods" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </label>
            <label class="field-group">
              <span>能量 · 恢复至 80%</span>
              <el-select v-model="selectedEnergyRecoveryMethod" size="large">
                <el-option v-for="item in recoveryMethods" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </label>
            <label class="field-group">
              <span>饮水 · 恢复至 80%</span>
              <el-select v-model="selectedHydrationRecoveryMethod" size="large">
                <el-option v-for="item in recoveryMethods" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </label>
          </div>
        </div>

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
              <span class="preset-card__summary">{{ presetLabel(index) }}</span>
            </button>
          </div>
        </div>

        <div v-if="shoppingNeeds().length" class="quest-block">
          <div class="recovery-block__heading">
            <span>本趟购物清单</span>
            <small>合同与藏身处升级只认局内带出</small>
          </div>
          <div v-for="need in shoppingNeeds()" :key="need.key" class="quest-row">
            <div>
              <strong>{{ need.detail }}</strong>
              <small>{{ need.label }} · {{ need.have }}/{{ need.need }}</small>
            </div>
            <span v-if="need.have >= need.need" class="quest-done">已齐</span>
            <span v-else class="quest-pending">待搜</span>
          </div>
        </div>

        <div class="quest-block">
          <div class="recovery-block__heading">
            <span>商人合同</span>
            <small>上交物品必须是撤离带出的，商店购买无效</small>
          </div>
          <div v-if="!visibleQuests().length" class="style-note">暂无合同</div>
          <div v-for="quest in visibleQuests()" :key="quest.id" class="quest-row">
            <div>
              <strong>{{ quest.name }}</strong>
              <small>{{ quest.merchantName }} · {{ quest.targetLabel }} {{ quest.current }}/{{ quest.required }} · ￥{{ quest.rewardCash }}</small>
            </div>
            <el-button v-if="quest.canAccept" size="small" :loading="acceptingId === quest.id" @click="emit('acceptQuest', quest.id)">接取</el-button>
            <el-button v-else-if="quest.canTurnIn" type="primary" size="small" :loading="turningId === quest.id" @click="emit('turninQuest', quest.id)">上交</el-button>
            <span v-else-if="quest.status === 'completed'" class="quest-done">已完成</span>
            <span v-else class="quest-pending">进行中</span>
          </div>
        </div>

        <div class="loadout-row">
          <div class="deployment-loadout">
            <span>当前携行</span>
            <strong>{{ currentWeapon?.name || '未装备武器' }} · {{ currentArmor?.name || '未装备护甲' }}</strong>
            <small v-if="hasCurrentWeapon || hasCurrentArmor">携带弹药：{{ carriedAmmoSummary() }}（弹药槽请在角色页面配置）</small>
            <small v-else>当前未装备任何装备，开局将按失能预案自动补购</small>
          </div>
        </div>
      </div>

      <aside class="launch-panel surface-panel">
        <div class="panel-heading">
          <div><span>02</span><h2>开始行动</h2></div>
          <small>配置确认后立即进入实时行动</small>
        </div>

        <div class="launch-summary">
          <div><span>目标区域</span><strong>{{ maps.find((item) => item.id === selectedMap)?.name || '未选择' }}</strong></div>
          <div><span>行动风格</span><strong>{{ styles.find((item) => item.value === selectedStyle)?.label }}</strong></div>
          <div><span>失能预案</span><strong>{{ presetLabel(selectedPreset) }}</strong></div>
          <div><span>生命恢复</span><strong>{{ recoveryMethodLabel(selectedHPRecoveryMethod) }}</strong></div>
          <div><span>携带弹药</span><strong>{{ carriedAmmoSummary() }}</strong></div>
        </div>

        <div class="launch-block">
          <p><span class="status-dot" />{{ recoveryPending ? '恢复目标尚未达成' : player.hp <= 0 || player.energy <= 0 || player.hydration <= 0 ? '行动员资源不足' : '行动员状态正常' }}</p>
          <el-button
            type="primary"
            size="large"
            :icon="VideoPlay"
            :loading="starting"
            :disabled="!canSubmit || recoveryPending || player.hp <= 0 || player.energy <= 0 || player.hydration <= 0"
            @click="startSession"
          >开始行动</el-button>
        </div>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.quest-row { display: flex; justify-content: space-between; gap: 12px; align-items: center; margin-top: 8px; }
.quest-row small { display: block; opacity: 0.72; }
.quest-done { font-size: 12px; color: var(--olive); }
.quest-pending { font-size: 12px; color: var(--amber); }
</style>
