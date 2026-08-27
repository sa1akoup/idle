<!-- 探索部署页：配置地图、行动风格与失能后的预设装备，并启动后台行动。 -->
<script setup lang="ts">
import { Compass, VideoPlay } from '@element-plus/icons-vue'
import { useExploreDeployment, type ExploreEmit, type ExploreProps } from '../composables/useExploreDeployment'

const props = defineProps<ExploreProps>()
const emit = defineEmits<ExploreEmit>()
const {
  styles, selectedMap, selectedStyle, selectedPreset, selectedAmmoId, selectedAmmoRounds, starting,
  recoveryMethods, selectedHPRecoveryMethod, selectedEnergyRecoveryMethod, selectedHydrationRecoveryMethod,
  currentWeapon, deploymentArmor, recoveryPending, compatibleOwnedAmmos, ammoInventoryQuantity, selectedAmmoStock,
  presetSummary, selectedPresetSummary, recoveryMethodLabel, canSubmit, startSession,
} = useExploreDeployment(props, emit)
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
              <span class="preset-card__summary">{{ presetSummary(index) }}</span>
            </button>
          </div>
          <small>当前所选：{{ selectedPresetSummary }}</small>
        </div>

        <div class="loadout-row">
          <div class="deployment-loadout">
            <span>当前携行</span>
            <strong>{{ currentWeapon?.name || '自动补购武器' }} · {{ deploymentArmor?.name || '自动补购护甲' }}</strong>
            <small>补给：{{ loadout.consumables.map((id) => consumables.find((item) => item.id === id)?.name ?? id).join('、') || '无' }}，装备配置请在角色页面调整</small>
          </div>
        </div>

        <div v-if="currentWeapon?.ammoPerRound" class="form-grid">
          <label class="field-group">
            <span>携带弹药</span>
            <el-select v-model="selectedAmmoId" size="large" placeholder="仓库中没有兼容弹药">
              <el-option
                v-for="ammo in compatibleOwnedAmmos"
                :key="ammo.id"
                :label="`${ammo.name} · 库存 ${ammoInventoryQuantity(ammo.id)} 发`"
                :value="ammo.id"
              />
            </el-select>
          </label>
          <label class="field-group">
            <span>携弹发数</span>
            <el-input-number
              v-model="selectedAmmoRounds"
              :min="currentWeapon.ammoPerRound"
              :max="Math.max(currentWeapon.ammoPerRound, selectedAmmoStock)"
              :step="currentWeapon.ammoPerRound"
              size="large"
              controls-position="right"
            />
            <small>启动后从仓库预留；本批弹药耗尽后，按 Session 启动时可购买的最高等级自动降级补给</small>
          </label>
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
          <div><span>失能预案</span><strong>预设 {{ selectedPreset }}</strong></div>
          <div><span>生命恢复</span><strong>{{ recoveryMethodLabel(selectedHPRecoveryMethod) }}</strong></div>
          <div v-if="currentWeapon?.ammoPerRound"><span>携带弹药</span><strong>{{ ammos.find((item) => item.id === selectedAmmoId)?.name || '未配置' }} ×{{ selectedAmmoRounds }}</strong></div>
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
