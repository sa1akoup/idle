<!-- 藏身处页：左侧模块列表（发电机+各设施），右侧按模块渲染详情；工作台提供制造/维修双标签。 -->
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue'
import { Box, CircleCheck, Download, FirstAidKit, House, Lock, Monitor, SwitchButton, Timer, Tools, Upload } from '@element-plus/icons-vue'
import type { Armor, ArmorInstance, Consumable, CraftingRecipe, HideoutFacility, HideoutJob, HideoutRequirement, HideoutSnapshot, HideoutUpgrade, ItemInstance, Player, StorageCapacity } from '../types'

const props = defineProps<{
  player: Player
  armors: Armor[]
  armorInstances: ArmorInstance[]
  itemInstances: ItemInstance[]
  consumables: Consumable[]
  craftingRecipes: CraftingRecipe[]
  repairingId: number | null
  craftingId: string | null
  storageCapacity: StorageCapacity | null
  hideout: HideoutSnapshot | null
  upgradingFacilityId: string | null
}>()

const emit = defineEmits<{
  repair: [id: number]
  upgrade: [facilityId: string]
  toggleGenerator: [enabled: boolean]
  loadGeneratorFuel: [instanceId: number]
  unloadGeneratorFuel: [instanceId: number]
  craft: [recipeId: string]
}>()

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined

const activeModule = ref('workbench')
const workbenchTab = ref<'craft' | 'repair'>('craft')

const facilityIcons: Record<string, Component> = {
  storage: Box,
  medical: FirstAidKit,
  workbench: Tools,
  intel: Monitor,
	booze_generator: FirstAidKit,
	bitcoin_farm: Monitor,
	scav_case: Box,
	shooting_range: Tools,
	gym: House,
	library: Monitor,
	hall_of_fame: House,
}

const facilities = computed(() => props.hideout?.facilities ?? [])
const activeJobs = computed(() => props.hideout?.jobs ?? [])
const activeRepairJobs = computed(() => activeJobs.value.filter((job) => job.jobType === 'repair'))
const facilityCount = computed(() => facilities.value.length)
const readyCount = computed(() => facilities.value.filter((facility) => facility.level > 0).length)
const generator = computed(() => props.hideout?.generator ?? null)
const fuelCandidates = computed(() => props.itemInstances.filter((instance) => {
	if (instance.locationType !== 'inventory' || instance.status !== 'normal') return false
	return props.consumables.some((item) => item.id === instance.itemId && item.fuelSeconds > 0)
}))
const activeFacility = computed(() => facilities.value.find((facility) => facility.id === activeModule.value) ?? null)
const craftFacilityIds = new Set(['workbench', 'medstation', 'nutrition_unit'])
const facilityRecipes = computed(() => props.craftingRecipes.filter((recipe) => recipe.facilityId === activeModule.value))
const showCraftList = computed(() => craftFacilityIds.has(activeModule.value) && (activeModule.value !== 'workbench' || workbenchTab.value === 'craft'))
const showRepairList = computed(() => activeModule.value === 'workbench' && workbenchTab.value === 'repair')

function selectModule(id: string) {
  activeModule.value = id
}

function handleGeneratorChange(value: boolean | string | number) {
	emit('toggleGenerator', Boolean(value))
}

onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function iconFor(facility: HideoutFacility): Component {
  return facilityIcons[facility.iconKey] ?? Lock
}

function effectText(facility: HideoutFacility): string {
  if (facility.runtime === 'planned') {
    return facility.level > 0 ? '规划中 · 已建成，本版本不生产' : '规划中 · 可升级储备材料'
  }
  if (facility.storageBonus > 0) return `仓库 +${facility.storageBonus} 格`
  if (facility.hpRecoveryPerHour > 0) return `生命 +${facility.hpRecoveryPerHour}/小时`
  if (facility.energyRecoveryPerHour > 0) return `能量 +${facility.energyRecoveryPerHour}/小时`
  if (facility.hydrationRecoveryPerHour > 0) return `饮水 +${facility.hydrationRecoveryPerHour}/小时`
  if (facility.stressRecoveryPerHour > 0) return `压力 −${facility.stressRecoveryPerHour}/小时`
  if (facility.repairKitDiscountPercent > 0) return `维修包消耗 -${facility.repairKitDiscountPercent}%`
  if (facility.repairSpeedPercent > 0) return `维修耗时 -${facility.repairSpeedPercent}%`
  if (facility.fuelConsumptionReductionPercent > 0) return `燃料消耗 -${facility.fuelConsumptionReductionPercent}%`
  if (facility.physicalSkillGrowthPercent > 0) return `身体技能成长 +${facility.physicalSkillGrowthPercent}%`
  if (facility.intelBonusPercent > 0) return `情报质量 +${facility.intelBonusPercent}%`
  return facility.level > 0 ? '基础模块已接通' : '尚未接入安全区网络'
}

function nextEffectText(facility: HideoutFacility): string {
  const next = facility.nextUpgrade
  if (!next) return '已达到当前最高等级'
  if (facility.id === 'storage') return `升级后仓库容量 +${next.level === 2 ? 120 : 240} 格`
  if (facility.id === 'medstation') return `升级后生命恢复 +${facility.nextUpgrade?.level === 2 ? '75' : '100'}/小时`
  if (facility.id === 'heating') return '升级后能量恢复效率提升'
  if (facility.id === 'water_collector') return '升级后饮水恢复效率提升'
  if (facility.id === 'rest_area') return '升级后压力恢复效率提升'
  if (facility.id === 'workbench') return '升级后维修更快、解锁更高等级制造'
  if (facility.id === 'nutrition_unit') return '升级后解锁口粮与饮水制造'
  if (facility.runtime === 'planned') return '升级消耗材料，生产能力将在后续版本接通'
  return '升级后情报质量提升'
}

function requirementValue(requirement: HideoutRequirement): string {
  if (Number.isInteger(requirement.requiredValue)) return String(requirement.requiredValue)
  return requirement.requiredValue.toFixed(1)
}

function requirementText(requirement: HideoutRequirement): string {
  if (requirement.requirementType === 'item') return `${requirement.label} ×${requirement.quantity}（局内带出）`
  if (requirement.requirementType === 'facility') return `${requirement.label} LV.${requirementValue(requirement)}`
  if (requirement.requirementType === 'trader') return `${requirement.label} 好感 ${requirementValue(requirement)}`
  return `${requirement.label} ${requirementValue(requirement)}`
}

function requirementSummary(upgrade: HideoutUpgrade): string {
  if (!upgrade.requirements.length) return upgrade.materialName ? `${upgrade.materialName} ×${upgrade.materialQuantity}` : '无额外条件'
  return upgrade.requirements.map(requirementText).join(' · ')
}

function jobForArmor(id: number): HideoutJob | undefined {
  return activeRepairJobs.value.find((job) => job.armorInstanceId === id)
}

function armorRemainingTime(id: number): string {
  const job = jobForArmor(id)
  return job ? remainingTime(job) : ''
}

function jobProgress(job: HideoutJob | undefined): number {
  if (!job) return 0
  const start = new Date(job.startedAt).getTime()
  const end = new Date(job.completeAt).getTime()
  if (end <= start) return 100
  return Math.max(0, Math.min(100, ((now.value - start) / (end - start)) * 100))
}

function remainingTime(job: HideoutJob): string {
  const seconds = Math.max(0, Math.ceil((new Date(job.completeAt).getTime() - now.value) / 1000))
  if (seconds < 60) return `${seconds} 秒`
  return `${Math.ceil(seconds / 60)} 分钟`
}

function armorName(instance: ArmorInstance): string {
  return props.armors.find((armor) => armor.id === instance.armorId)?.name ?? instance.armorId
}

function durabilityPercent(instance: ArmorInstance): number {
  if (instance.maxDurability <= 0) return 0
  return Math.max(0, Math.min(100, Math.round((instance.curDurability / instance.maxDurability) * 100)))
}

function repairButtonText(instance: ArmorInstance): string {
  if (instance.status === 'repairing') return '维修中'
  if (instance.repairCount >= 1) return '已达上限'
  if (instance.curDurability > 0) return '状态正常'
  return `排入队列 · ￥${props.hideout?.repairCost ?? 0}`
}

function canRepair(instance: ArmorInstance): boolean {
  return instance.curDurability <= 0 && instance.repairCount < 1 && instance.status !== 'repairing'
}

function itemName(itemID: string): string {
  return props.consumables.find((item) => item.id === itemID)?.name ?? itemID
}

function fuelPercent(current: number, max: number): number {
  return max > 0 ? Math.max(0, Math.min(100, Math.round(current / max * 100))) : 0
}

function fuelTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  return minutes >= 60 ? `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分钟` : `${minutes} 分钟`
}

function materialClass(input: { satisfied: boolean }): string {
  return input.satisfied ? 'satisfied' : 'missing'
}

function recipeProgress(recipe: CraftingRecipe): number {
  if (!recipe.inputs.length) return recipe.canStart ? 100 : 0
  const satisfied = recipe.inputs.filter((input) => input.satisfied).length
  return Math.round((satisfied / recipe.inputs.length) * 100)
}
</script>

<template>
  <section class="view-page hideout-view">
    <header class="page-heading hideout-heading">
      <div>
        <span class="eyebrow">SECURE ZONE / MODULE CONTROL</span>
        <h1>藏身处</h1>
        <p>把每次撤离带回来的材料，换成下一次行动的准备度。</p>
      </div>
      <div class="hideout-heading__status">
        <span class="status-dot" />
        <div><small>安全区状态</small><strong>{{ props.player.hp <= 0 || props.player.energy <= 0 || props.player.hydration <= 0 ? '资源恢复中' : '基础设施在线' }}</strong></div>
      </div>
    </header>

    <section class="hideout-overview">
      <div class="hideout-overview__lead">
        <span class="eyebrow">SAFEHOUSE STATUS</span>
        <strong>{{ readyCount }} / {{ facilityCount }} 模块已接入</strong>
        <small>{{ activeJobs.length ? `${activeJobs.length} 项作业正在执行` : '当前没有排队作业' }}</small>
      </div>
      <div class="hideout-stat">
        <span><el-icon><Box /></el-icon>仓库容量</span>
        <strong>{{ storageCapacity?.used ?? props.hideout?.storageCapacity.used ?? 0 }} <em>/ {{ storageCapacity?.capacity ?? props.hideout?.storageCapacity.capacity ?? 0 }}</em></strong>
      </div>
      <div class="hideout-stat">
        <span><el-icon><FirstAidKit /></el-icon>伤势恢复</span>
        <strong>+{{ props.hideout?.bonuses.hpRecoveryPerHour ?? 0 }}<em> HP / 小时</em></strong>
      </div>
      <div class="hideout-stat">
        <span><el-icon><Tools /></el-icon>维修成本</span>
        <strong>￥{{ props.hideout?.repairCost ?? 0 }}<em> / 次</em></strong>
      </div>
      <div class="hideout-stat">
        <span><el-icon><Timer /></el-icon>运行作业</span>
        <strong>{{ activeJobs.length }}<em> 项</em></strong>
      </div>
    </section>

    <div class="hideout-shell">
      <aside class="hideout-module-list surface-panel">
        <div class="panel-heading">
          <div><span>MODULE LIST / {{ String(facilityCount + 1).padStart(2, '0') }}</span><h2>模块列表</h2></div>
          <el-icon><House /></el-icon>
        </div>
        <button
          v-if="generator"
          type="button"
          class="module-list-item"
          :class="{ active: activeModule === 'generator' }"
          @click="selectModule('generator')"
        >
          <span class="facility-card__icon"><el-icon><SwitchButton /></el-icon></span>
          <div class="module-list-item__copy">
            <small>POWER SYSTEM</small>
            <strong>发电机</strong>
            <em>{{ generator.enabled ? '运行中' : '待机' }}</em>
          </div>
          <b>{{ generator.fuels.length }}<em>/{{ generator.fuelSlots }}</em></b>
        </button>
        <button
          v-for="facility in facilities"
          :key="facility.id"
          type="button"
          class="module-list-item"
          :class="{ active: activeModule === facility.id, locked: facility.level === 0, upgrading: facility.state === 'upgrading' }"
          @click="selectModule(facility.id)"
        >
          <span class="facility-card__icon"><el-icon><component :is="iconFor(facility)" /></el-icon></span>
          <div class="module-list-item__copy">
            <small>{{ facility.category.toUpperCase() }}</small>
            <strong>{{ facility.name }}</strong>
            <em>{{ effectText(facility) }}</em>
          </div>
          <b>LV.{{ facility.level }}<em>/{{ facility.maxLevel }}</em></b>
        </button>
      </aside>

      <section class="hideout-module-detail surface-panel">
        <div class="panel-heading">
          <div><span>MODULE VIEW</span><h2>{{ activeFacility?.name ?? '发电机' }}</h2></div>
          <el-icon><component :is="activeModule === 'generator' ? SwitchButton : activeFacility ? iconFor(activeFacility) : Lock" /></el-icon>
        </div>

        <template v-if="activeModule === 'generator' && generator">
          <div class="generator-summary">
            <span>{{ generator.enabled ? '运行中' : '待机' }}</span>
            <strong>剩余 {{ fuelTime(generator.fuelRemainingSeconds) }}</strong>
            <small>{{ generator.fuels.length }} / {{ generator.fuelSlots }} 个燃料槽</small>
            <el-switch :model-value="generator.enabled" :active-icon="SwitchButton" :disabled="!generator.fuels.length" @change="handleGeneratorChange" />
          </div>
          <div class="generator-fuels">
            <div v-for="fuel in generator.fuels" :key="fuel.instanceId" class="generator-fuel-row">
              <div><strong>{{ itemName(fuel.itemId) }}</strong><small>实例 #{{ fuel.instanceId }} · {{ fuelTime(fuel.fuelSeconds) }}</small></div>
              <div class="generator-fuel-row__durability"><span>{{ fuelPercent(fuel.currentDurability, fuel.maxDurability) }}%</span><el-progress :percentage="fuelPercent(fuel.currentDurability, fuel.maxDurability)" :show-text="false" /></div>
              <el-button :icon="Download" circle title="卸载燃料" @click="emit('unloadGeneratorFuel', fuel.instanceId)" />
            </div>
            <div v-for="fuel in fuelCandidates" :key="fuel.id" class="generator-fuel-row generator-fuel-row--available">
              <div><strong>{{ itemName(fuel.itemId) }}</strong><small>仓库实例 #{{ fuel.id }}</small></div>
              <span class="generator-fuel-row__durability">{{ fuelPercent(fuel.currentDurability, fuel.maxDurability) }}%</span>
              <el-button :icon="Upload" circle title="装载燃料" :disabled="generator.fuels.length >= generator.fuelSlots" @click="emit('loadGeneratorFuel', fuel.id)" />
            </div>
            <div v-if="!generator.fuels.length && !fuelCandidates.length" class="generator-empty">仓库中暂无可用燃料实例</div>
          </div>
        </template>

        <template v-else-if="activeFacility && craftFacilityIds.has(activeFacility.id)">
          <div v-if="activeFacility.state === 'upgrading'" class="module-detail__progress">
            <span><b>施工中</b><em>等待模块上线</em></span>
            <el-progress :percentage="jobProgress(activeJobs.find((job) => job.facilityId === activeFacility?.id))" :show-text="false" />
          </div>
          <div v-else-if="activeFacility.nextUpgrade" class="module-detail__upgrade">
            <div class="module-detail__upgrade-copy">
              <span>下一阶段 · LV.{{ activeFacility.nextUpgrade.level }}</span>
              <strong>{{ nextEffectText(activeFacility) }}</strong>
              <small>￥{{ activeFacility.nextUpgrade.cost }} · {{ Math.ceil(activeFacility.nextUpgrade.durationSec / 60) || 1 }} 分钟</small>
              <small class="module-detail__requirements">{{ requirementSummary(activeFacility.nextUpgrade) }}</small>
            </div>
            <el-button
              :icon="Tools"
              :loading="props.upgradingFacilityId === activeFacility.id"
              :disabled="!activeFacility.nextUpgrade.canStart"
              @click="emit('upgrade', activeFacility.id)"
            >升级</el-button>
          </div>
          <el-tabs v-if="activeFacility.id === 'workbench'" v-model="workbenchTab" class="workbench-tabs">
            <el-tab-pane label="制造" name="craft" />
            <el-tab-pane label="维修队列" name="repair" />
          </el-tabs>
          <div v-if="showCraftList" class="crafting-list">
                <article
                  v-for="recipe in facilityRecipes"
                  :key="recipe.id"
                  class="crafting-row"
                  :class="{ locked: !recipe.canStart }"
                >
                  <div class="crafting-row__head">
                    <div>
                      <small>REQ. {{ recipe.facilityName || '设施' }} LV.{{ recipe.requiredLevel }} · {{ recipe.craftMinutes }} 分钟</small>
                      <strong>{{ recipe.name }}</strong>
                    </div>
                    <el-progress :percentage="recipeProgress(recipe)" :show-text="false" />
                  </div>
                  <div class="crafting-row__body">
                    <div class="crafting-materials">
                      <span class="crafting-materials__label">材料</span>
                      <span v-for="input in recipe.inputs" :key="input.itemId" class="crafting-material" :class="materialClass(input)">
                        {{ input.name }}（局内带出） <b>{{ input.have }} / {{ input.need }}</b>
                      </span>
                    </div>
                    <div class="crafting-output">
                      <span class="crafting-materials__label">产物</span>
                      <span class="crafting-output__item" :class="{ instance: recipe.output.instanceRequired }">
                        {{ recipe.output.name }} ×{{ recipe.output.quantity }}<em v-if="recipe.output.instanceRequired">耐久成品</em>
                      </span>
                    </div>
                  </div>
                  <div class="crafting-row__foot">
                    <span v-if="!recipe.canStart">{{ recipe.reason }}</span>
                    <span v-else>消耗材料后即刻开工</span>
                    <el-button
                      type="primary"
                      size="small"
                      :icon="Tools"
                      :loading="props.craftingId === recipe.id"
                      :disabled="!recipe.canStart"
                      @click="emit('craft', recipe.id)"
                    >制造</el-button>
                  </div>
                </article>
                <div v-if="!facilityRecipes.length" class="crafting-empty">暂无可用配方</div>
              </div>
          <div v-if="showRepairList">
              <div class="repair-bench__meta">维修归零护甲 · 完成后耐久上限减半</div>
              <div v-if="props.armorInstances.length" class="repair-list">
                <div v-for="instance in props.armorInstances" :key="instance.id" class="repair-row">
                  <div class="repair-row__identity"><span class="item-icon"><el-icon><Tools /></el-icon></span><div><strong>{{ armorName(instance) }}</strong><small>维修记录 {{ instance.repairCount }} / 1</small></div></div>
                  <div class="repair-durability"><span>{{ instance.curDurability }} / {{ instance.maxDurability }}</span><el-progress :percentage="durabilityPercent(instance)" :show-text="false" /></div>
                  <div class="repair-row__status">
                    <span v-if="jobForArmor(instance.id)" class="repair-status repairing"><i class="status-dot warning" />{{ armorRemainingTime(instance.id) }}</span>
                    <span v-else-if="instance.status === 'broken'" class="repair-status broken"><i class="status-dot danger" />待维修</span>
                    <span v-else class="repair-status"><i class="status-dot" />可用</span>
                  </div>
                  <el-button :icon="Tools" :loading="props.repairingId === instance.id" :disabled="!canRepair(instance)" @click="emit('repair', instance.id)">{{ repairButtonText(instance) }}</el-button>
                </div>
              </div>
              <div v-else class="repair-empty"><el-icon><Box /></el-icon><span>暂无护甲实例</span></div>
          </div>
        </template>

        <template v-else-if="activeFacility">
          <p class="module-detail__description">{{ activeFacility.description }}</p>
          <div class="module-detail__effect">
            <span>当前效果</span>
            <strong>{{ effectText(activeFacility) }}</strong>
          </div>
          <div v-if="activeFacility.state === 'upgrading'" class="module-detail__progress">
            <span><b>施工中</b><em>等待模块上线</em></span>
            <el-progress :percentage="jobProgress(activeJobs.find((job) => job.facilityId === activeFacility?.id))" :show-text="false" />
          </div>
          <div v-else-if="activeFacility.nextUpgrade" class="module-detail__upgrade">
            <div class="module-detail__upgrade-copy">
              <span>下一阶段 · LV.{{ activeFacility.nextUpgrade.level }}</span>
              <strong>{{ nextEffectText(activeFacility) }}</strong>
              <small>￥{{ activeFacility.nextUpgrade.cost }} · {{ Math.ceil(activeFacility.nextUpgrade.durationSec / 60) || 1 }} 分钟</small>
              <small class="module-detail__requirements">{{ requirementSummary(activeFacility.nextUpgrade) }}</small>
            </div>
            <el-button
              :icon="Tools"
              :loading="props.upgradingFacilityId === activeFacility.id"
              :disabled="!activeFacility.nextUpgrade.canStart"
              @click="emit('upgrade', activeFacility.id)"
            >升级</el-button>
          </div>
          <div v-else class="module-detail__max"><el-icon><CircleCheck /></el-icon><span>最高等级模块</span></div>
        </template>

        <div v-if="activeJobs.length" class="module-jobs">
          <div class="panel-heading">
            <div><span>ACTIVE QUEUE / {{ String(activeJobs.length).padStart(2, '0') }}</span><h2>作业队列</h2></div>
            <el-icon><Timer /></el-icon>
          </div>
          <div class="job-list">
            <article v-for="job in activeJobs" :key="job.id" class="job-card">
              <div class="job-card__head">
                <span class="job-card__icon"><el-icon><component :is="job.jobType === 'repair' ? Tools : iconFor(facilities.find((item) => item.id === job.facilityId) ?? ({} as HideoutFacility))" /></el-icon></span>
                <div><small>{{ job.jobType === 'repair' ? 'SERVICE' : job.jobType === 'craft' ? 'CRAFTING' : 'CONSTRUCTION' }}</small><strong>{{ job.jobType === 'repair' ? '护甲维修' : job.jobType === 'craft' ? facilities.find((item) => item.id === job.facilityId)?.name : facilities.find((item) => item.id === job.facilityId)?.name }}{{ job.jobType === 'upgrade' ? ` · LV.${job.targetLevel}` : '' }}</strong></div>
                <b>{{ remainingTime(job) }}</b>
              </div>
              <el-progress :percentage="jobProgress(job)" :show-text="false" />
              <div class="job-card__foot"><span>{{ Math.round(jobProgress(job)) }}% 已完成</span><em>完成于 {{ new Date(job.completeAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</em></div>
            </article>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>