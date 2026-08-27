<!-- 藏身处页：以模块网格、升级队列和维修台展示玩家的局外养成状态。 -->
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue'
import { Box, CircleCheck, Download, FirstAidKit, House, Lock, Monitor, SwitchButton, Timer, Tools, Upload } from '@element-plus/icons-vue'
import type { Armor, ArmorInstance, Consumable, HideoutFacility, HideoutJob, HideoutRequirement, HideoutSnapshot, HideoutUpgrade, ItemInstance, Player, StorageCapacity } from '../types'

const props = defineProps<{
  player: Player
  armors: Armor[]
  armorInstances: ArmorInstance[]
  itemInstances: ItemInstance[]
  consumables: Consumable[]
  repairingId: number | null
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
}>()

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined

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

const activeJobs = computed(() => props.hideout?.jobs ?? [])
const activeRepairJobs = computed(() => activeJobs.value.filter((job) => job.jobType === 'repair'))
const facilityCount = computed(() => props.hideout?.facilities.length ?? 0)
const readyCount = computed(() => props.hideout?.facilities.filter((facility) => facility.level > 0).length ?? 0)
const generator = computed(() => props.hideout?.generator ?? null)
const fuelCandidates = computed(() => props.itemInstances.filter((instance) => {
	if (instance.locationType !== 'inventory' || instance.status !== 'normal') return false
	return props.consumables.some((item) => item.id === instance.itemId && item.fuelSeconds > 0)
}))

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
  if (facility.storageBonus > 0) return `仓库 +${facility.storageBonus} 格`
  if (facility.hpRecoveryPerHour > 0) return `生命 +${facility.hpRecoveryPerHour}/小时`
  if (facility.energyRecoveryPerHour > 0) return `能量 +${facility.energyRecoveryPerHour}/小时`
  if (facility.hydrationRecoveryPerHour > 0) return `饮水 +${facility.hydrationRecoveryPerHour}/小时`
  if (facility.repairKitDiscountPercent > 0) return `维修包消耗 -${facility.repairKitDiscountPercent}%`
  if (facility.repairSpeedPercent > 0) return `维修耗时 -${facility.repairSpeedPercent}%`
  if (facility.fuelConsumptionReductionPercent > 0) return `燃料消耗 -${facility.fuelConsumptionReductionPercent}%`
  if (facility.physicalSkillGrowthPercent > 0) return `身体技能成长 +${facility.physicalSkillGrowthPercent}%`
  if (facility.intelBonusPercent > 0) return `情报质量 +${facility.intelBonusPercent}%`
	if (facility.id === 'booze_generator' && facility.level > 0) return '可生产月光酒'
	if (facility.id === 'bitcoin_farm' && facility.level > 0) return '可生产实物比特币'
	if (facility.id === 'scav_case' && facility.level > 0) return '可派遣 Scav 搜集物资'
	if (facility.id === 'shooting_range' && facility.level > 0) return '可进行射击训练'
	if (facility.id === 'gym' && facility.level > 0) return '可进行身体技能训练'
	if (facility.id === 'library' && facility.level > 0) return '行动经验与技能成长提升'
	if (facility.id === 'hall_of_fame' && facility.level > 0) return '荣誉陈列与技能成长提升'
  return facility.level > 0 ? '基础模块已接通' : '尚未接入安全区网络'
}

function nextEffectText(facility: HideoutFacility): string {
  const next = facility.nextUpgrade
  if (!next) return '已达到当前最高等级'
  if (facility.id === 'storage') return `升级后仓库容量 +${next.level === 2 ? 20 : 40} 格`
  if (facility.id === 'medstation') return `升级后生命恢复 +${facility.nextUpgrade?.level === 2 ? '82.2' : '173.5'}/小时`
  if (facility.id === 'heating') return '升级后能量恢复效率提升'
  if (facility.id === 'water_collector') return '升级后饮水恢复效率提升'
  if (facility.id === 'workbench') return '升级后维修作业更快完成'
	if (facility.id === 'booze_generator') return '升级后提高月光酒生产能力'
	if (facility.id === 'bitcoin_farm') return '升级后提高实物比特币生产能力'
	if (facility.id === 'scav_case') return '升级后提高 Scav 派遣收益'
	if (facility.id === 'shooting_range') return '升级后解锁更高等级训练'
	if (facility.id === 'gym') return '升级后提高身体技能训练能力'
	if (facility.id === 'library') return '升级后提高行动经验和技能成长'
	if (facility.id === 'hall_of_fame') return '升级后提高荣誉陈列与技能成长'
  return '升级后情报质量提升'
}

function requirementValue(requirement: HideoutRequirement): string {
  if (Number.isInteger(requirement.requiredValue)) return String(requirement.requiredValue)
  return requirement.requiredValue.toFixed(1)
}

function requirementText(requirement: HideoutRequirement): string {
  if (requirement.requirementType === 'item') return `${requirement.label} ×${requirement.quantity}`
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

    <section v-if="generator" class="generator-panel surface-panel">
      <div class="panel-heading">
        <div><span>POWER SYSTEM</span><h2>发电机</h2></div>
		<el-switch :model-value="generator.enabled" :active-icon="SwitchButton" :disabled="!generator.fuels.length" @change="handleGeneratorChange" />
      </div>
      <div class="generator-summary">
        <span>{{ generator.enabled ? '运行中' : '待机' }}</span>
        <strong>剩余 {{ fuelTime(generator.fuelRemainingSeconds) }}</strong>
        <small>{{ generator.fuels.length }} / {{ generator.fuelSlots }} 个燃料槽</small>
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
    </section>

    <div class="hideout-layout">
      <section class="facility-board surface-panel">
        <div class="panel-heading">
          <div><span>MODULE GRID / {{ String(facilityCount).padStart(2, '0') }}</span><h2>设施模块</h2></div>
          <el-icon><House /></el-icon>
        </div>
        <div class="facility-grid">
          <article
            v-for="facility in props.hideout?.facilities ?? []"
            :key="facility.id"
            class="facility-card"
            :class="{ locked: facility.level === 0, upgrading: facility.state === 'upgrading' }"
          >
            <div class="facility-card__head">
              <span class="facility-card__icon"><el-icon><component :is="iconFor(facility)" /></el-icon></span>
              <div>
                <small>{{ facility.category.toUpperCase() }}</small>
                <strong>{{ facility.name }}</strong>
              </div>
              <b>LV.{{ facility.level }}<em>/{{ facility.maxLevel }}</em></b>
            </div>
            <p class="facility-card__description">{{ facility.description }}</p>
            <div class="facility-card__effect">
              <span>当前效果</span>
              <strong>{{ effectText(facility) }}</strong>
            </div>
            <div v-if="facility.state === 'upgrading'" class="facility-card__progress">
              <span><b>施工中</b><em>等待模块上线</em></span>
              <el-progress :percentage="jobProgress(activeJobs.find((job) => job.facilityId === facility.id))" :show-text="false" />
            </div>
            <div v-else-if="facility.nextUpgrade" class="facility-card__upgrade">
              <div>
                <span>下一阶段 · LV.{{ facility.nextUpgrade.level }}</span>
                <strong>{{ nextEffectText(facility) }}</strong>
					<small>￥{{ facility.nextUpgrade.cost }} · {{ Math.ceil(facility.nextUpgrade.durationSec / 60) || 1 }} 分钟</small>
					<small class="facility-card__requirements">{{ requirementSummary(facility.nextUpgrade) }}</small>
              </div>
              <el-button
                :icon="Tools"
                :loading="props.upgradingFacilityId === facility.id"
                :disabled="!facility.nextUpgrade.canStart"
                @click="emit('upgrade', facility.id)"
              >升级</el-button>
            </div>
            <div v-else class="facility-card__max"><el-icon><CircleCheck /></el-icon><span>最高等级模块</span></div>
          </article>
        </div>
      </section>

      <aside class="hideout-queue surface-panel">
        <div class="panel-heading">
          <div><span>ACTIVE QUEUE / {{ String(activeJobs.length).padStart(2, '0') }}</span><h2>作业队列</h2></div>
          <el-icon><Timer /></el-icon>
        </div>
        <div v-if="activeJobs.length" class="job-list">
          <article v-for="job in activeJobs" :key="job.id" class="job-card">
            <div class="job-card__head">
              <span class="job-card__icon"><el-icon><component :is="job.jobType === 'repair' ? Tools : iconFor(props.hideout?.facilities.find((item) => item.id === job.facilityId) ?? ({} as HideoutFacility))" /></el-icon></span>
              <div><small>{{ job.jobType === 'repair' ? 'SERVICE' : 'CONSTRUCTION' }}</small><strong>{{ job.jobType === 'repair' ? '护甲维修' : props.hideout?.facilities.find((item) => item.id === job.facilityId)?.name }}{{ job.jobType === 'upgrade' ? ` · LV.${job.targetLevel}` : '' }}</strong></div>
              <b>{{ remainingTime(job) }}</b>
            </div>
            <el-progress :percentage="jobProgress(job)" :show-text="false" />
            <div class="job-card__foot"><span>{{ Math.round(jobProgress(job)) }}% 已完成</span><em>完成于 {{ new Date(job.completeAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</em></div>
          </article>
        </div>
        <div v-else class="queue-empty"><el-icon><Timer /></el-icon><strong>队列空闲</strong><span>选择一个模块开始建设</span></div>
      </aside>
    </div>

    <section class="repair-bench surface-panel">
      <div class="panel-heading">
        <div><span>SERVICE BAY / ARMOR</span><h2>护甲维修台</h2></div>
        <div class="repair-bench__meta">维修归零护甲 · 完成后耐久上限减半</div>
      </div>
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
    </section>
  </section>
</template>
