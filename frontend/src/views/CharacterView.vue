<!-- 角色页：展示玩家属性，并以 RPG 部位格子管理当前装备与失能后补购预设。 -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, EditPen, User } from '@element-plus/icons-vue'
import type {
  Armor, Backpack, ChestRig, Consumable, Headset, Helmet,
  InventoryItem, Player, PlayerLoadout, SaveLoadoutRequest, Weapon,
} from '../types'

const props = defineProps<{
  player: Player
  loadout: PlayerLoadout
  inventory: InventoryItem[]
  weapons: Weapon[]
  armors: Armor[]
  consumables: Consumable[]
  chestRigs: ChestRig[]
  backpacks: Backpack[]
  helmets: Helmet[]
  headsets: Headset[]
  savingName: boolean
  savingLoadout: boolean
}>()

const emit = defineEmits<{
  saveName: [name: string]
  saveLoadout: [request: SaveLoadoutRequest, silent?: boolean]
}>()

type SlotKey = 'weaponId' | 'armorId' | 'chestRigId' | 'backpackId' | 'helmetId' | 'headsetId'

interface SlotDef { key: SlotKey; label: string; row: 1 | 2; col: 1 | 2 | 3 }
type EquipSet = { [k in SlotKey]: string } & { consumables: string[]; name: string }

const slotDefs: SlotDef[] = [
  { key: 'headsetId', label: '耳机', row: 1, col: 1 },
  { key: 'helmetId', label: '头盔', row: 1, col: 2 },
  { key: 'chestRigId', label: '胸挂', row: 1, col: 3 },
  { key: 'weaponId', label: '武器', row: 2, col: 1 },
  { key: 'armorId', label: '护甲', row: 2, col: 2 },
  { key: 'backpackId', label: '背包', row: 2, col: 3 },
]
const gridRows = [1, 2] as const

const editing = ref(false)
const nameDraft = ref(props.player.name)

function emptySet(name = ''): EquipSet {
  return { weaponId: '', armorId: '', chestRigId: '', backpackId: '', helmetId: '', headsetId: '', consumables: [], name }
}

const current = ref<EquipSet>(emptySet())
const presets = ref<EquipSet[]>([emptySet('标准突击'), emptySet('轻装渗透'), emptySet('重装攻坚')])
const presetIndex = ref(0)

function fromLoadout(l: PlayerLoadout): EquipSet {
  return {
    weaponId: l.weaponId, armorId: l.armorId, chestRigId: l.chestRigId, backpackId: l.backpackId,
    helmetId: l.helmetId, headsetId: l.headsetId, consumables: [...l.consumables], name: '',
  }
}
function presetFromLoadout(l: PlayerLoadout, index: number): EquipSet {
  if (index === 0) {
    return {
      weaponId: l.presetWeaponId, armorId: l.presetArmorId, chestRigId: l.presetChestRigId, backpackId: l.presetBackpackId,
      helmetId: l.presetHelmetId, headsetId: l.presetHeadsetId, consumables: [...l.presetConsumables], name: l.presetName,
    }
  }
  if (index === 1) {
    return {
      weaponId: l.preset2WeaponId, armorId: l.preset2ArmorId, chestRigId: l.preset2ChestRigId, backpackId: l.preset2BackpackId,
      helmetId: l.preset2HelmetId, headsetId: l.preset2HeadsetId, consumables: [...l.preset2Consumables], name: l.preset2Name,
    }
  }
  return {
    weaponId: l.preset3WeaponId, armorId: l.preset3ArmorId, chestRigId: l.preset3ChestRigId, backpackId: l.preset3BackpackId,
    helmetId: l.preset3HelmetId, headsetId: l.preset3HeadsetId, consumables: [...l.preset3Consumables], name: l.preset3Name,
  }
}

// 从 loadout 同步状态期间跳过自动保存
let syncing = false
watch(() => props.loadout, (l) => {
  syncing = true
  current.value = fromLoadout(l)
  presets.value = [presetFromLoadout(l, 0), presetFromLoadout(l, 1), presetFromLoadout(l, 2)]
  syncing = false
}, { deep: true, immediate: true })
watch(() => props.player.name, (name) => { nameDraft.value = name })

const mainAttributes = computed(() => [
  { label: '力量', value: props.player.strength, code: 'STR' },
  { label: '敏捷', value: props.player.agility, code: 'AGI' },
  { label: '智力', value: props.player.intellect, code: 'INT' },
  { label: '魅力', value: props.player.charisma, code: 'CHA' },
])

const skills = computed(() => [
  ['潜行', props.player.stealth], ['感知', props.player.perception], ['交涉', props.player.negotiation], ['幸运', props.player.luck],
  ['生存', props.player.survival], ['抗压', props.player.resist], ['工程', props.player.engineering], ['医疗', props.player.medical],
] as const)

const proficiencies = computed(() => [
  ['近战', props.player.meleeProf], ['手枪', props.player.pistolProf], ['冲锋枪', props.player.smgProf],
  ['霰弹枪', props.player.shotgunProf], ['突击步枪', props.player.rifleProf], ['狙击枪', props.player.sniperProf],
] as const)

const ownedItemIds = computed(() => new Set(props.inventory.filter((item) => item.itemId !== 'cash' && item.quantity > 0).map((item) => item.itemId)))

// 归一化的部位候选（完整目录），ownedOnly=true 时仅取仓库内已有
function repTag(req: number): string {
  return req > 0 ? ` · 好感度 ${req}` : ''
}
function slotOptions(key: SlotKey): { id: string; name: string; detail: string }[] {
  const map: Record<SlotKey, { name: string; detail: string }[]> = {
    weaponId: props.weapons.map((i) => ({ name: i.name, detail: `伤害 ${i.damage} · 穿透 ${i.penetration} · ${i.weight}kg${repTag(i.repRequirement)}` })),
    armorId: props.armors.map((i) => ({ name: i.name, detail: `防护 ${i.protect} · 覆盖 ${i.coverage}% · ${i.weight}kg${repTag(i.repRequirement)}` })),
    helmetId: props.helmets.map((i) => ({ name: i.name, detail: `防护 ${i.protect} · 覆盖 ${i.coverage}% · ${i.weight}kg${repTag(i.repRequirement)}` })),
    headsetId: props.headsets.map((i) => ({ name: i.name, detail: `听力 Lv.${i.hearingLevel} · ${i.weight}kg${repTag(i.repRequirement)}` })),
    chestRigId: props.chestRigs.map((i) => ({ name: i.name, detail: `格数+${i.addSlots} 负重+${i.addWeight}kg · ${i.weight}kg${repTag(i.repRequirement)}` })),
    backpackId: props.backpacks.map((i) => ({ name: i.name, detail: `格数+${i.addSlots} 负重+${i.addWeight}kg · ${i.weight}kg${repTag(i.repRequirement)}` })),
  }
  const idMap: Record<SlotKey, string[]> = {
    weaponId: props.weapons.map((i) => i.id), armorId: props.armors.map((i) => i.id), helmetId: props.helmets.map((i) => i.id),
    headsetId: props.headsets.map((i) => i.id), chestRigId: props.chestRigs.map((i) => i.id), backpackId: props.backpacks.map((i) => i.id),
  }
  return idMap[key].map((id, idx) => ({ id, ...map[key][idx] }))
}

// 当前装备部位仅可从仓库（owned）选择
function currentOptions(key: SlotKey) {
  return slotOptions(key).filter((i) => ownedItemIds.value.has(i.id))
}

function activeSet(context: 'current' | 'preset'): EquipSet {
  return context === 'current' ? current.value : presets.value[presetIndex.value]
}
function slotId(context: 'current' | 'preset', key: SlotKey): string {
  return activeSet(context)[key]
}
function slotName(context: 'current' | 'preset', key: SlotKey): string {
  const id = slotId(context, key)
  if (!id) return ''
  return slotOptions(key).find((i) => i.id === id)?.name ?? ''
}

const pickerOpen = ref(false)
const pickerKind = ref<'equip' | 'consumable'>('equip')
const pickerContext = ref<'current' | 'preset'>('current')
const pickerKey = ref<SlotKey>('weaponId')
const pickerConsumableIndex = ref(0)

function openPicker(context: 'current' | 'preset', key: SlotKey) {
  pickerKind.value = 'equip'
  pickerContext.value = context
  pickerKey.value = key
  pickerOpen.value = true
}
function openConsumablePicker(context: 'current' | 'preset', index: number) {
  pickerKind.value = 'consumable'
  pickerContext.value = context
  pickerConsumableIndex.value = index
  pickerOpen.value = true
}
function pickOption(id: string) {
  const set = activeSet(pickerContext.value)
  if (pickerKind.value === 'consumable') {
    const arr = padConsumables(set.consumables)
    arr[pickerConsumableIndex.value] = id
    set.consumables = arr.filter(Boolean)
  } else {
    set[pickerKey.value] = id
  }
  pickerOpen.value = false
}
const pickerList = computed(() => {
  if (pickerKind.value === 'consumable') {
    return props.consumables.map((c) => ({ id: c.id, name: c.name, detail: `${c.desc} · ${c.weight}kg` }))
  }
  return pickerContext.value === 'current' ? currentOptions(pickerKey.value) : slotOptions(pickerKey.value)
})
const pickerTitle = computed(() => {
  if (pickerKind.value === 'consumable') {
    return `${pickerContext.value === 'current' ? '随身' : '预设'}补给 · ${pickerContext.value === 'current' ? '仓库道具' : '商人道具'}`
  }
  return `${slotDefs.find((s) => s.key === pickerKey.value)?.label} · ${pickerContext.value === 'current' ? '仓库装备' : '商人装备'}`
})

// 补给栏位：固定 4 格，空位以 '' 占位
const consumableSlotCount = 4
function padConsumables(arr: string[]): string[] {
  const out = Array(consumableSlotCount).fill('')
  arr.forEach((id, i) => { if (i < consumableSlotCount) out[i] = id })
  return out
}
function consumableAt(set: EquipSet, index: number): string {
  const id = padConsumables(set.consumables)[index]
  return props.consumables.find((c) => c.id === id)?.name ?? ''
}

// 实时携行容量：随当前选中装备与补给即时计算（与后端公式一致）
const liveCapacity = computed(() => {
  const strength = props.player.strength
  const baseSlots = 20
  const baseWeight = 50 + (strength - 50) * 0.3
  const chestRig = props.chestRigs.find((i) => i.id === current.value.chestRigId)
  const backpack = props.backpacks.find((i) => i.id === current.value.backpackId)
  const bonusSlots = (chestRig?.addSlots ?? 0) + (backpack?.addSlots ?? 0)
  const bonusWeight = (chestRig?.addWeight ?? 0) + (backpack?.addWeight ?? 0)

  const items: { weight: number; slots: number }[] = []
  const push = (id: string, list: { id: string; weight: number; slots: number }[]) => {
    const it = list.find((x) => x.id === id)
    if (it) items.push(it)
  }
  push(current.value.weaponId, props.weapons)
  push(current.value.armorId, props.armors)
  push(current.value.helmetId, props.helmets)
  push(current.value.headsetId, props.headsets)
  push(current.value.chestRigId, props.chestRigs)
  push(current.value.backpackId, props.backpacks)
  for (const id of current.value.consumables) push(id, props.consumables)

  const usedWeight = items.reduce((s, i) => s + i.weight, 0)
  const usedSlots = items.reduce((s, i) => s + i.slots, 0)
  return {
    baseSlots, bonusSlots, totalSlots: baseSlots + bonusSlots,
    baseWeight, bonusWeight, totalWeight: baseWeight + bonusWeight,
    usedSlots, usedWeight,
  }
})

function submitName() {
  const name = nameDraft.value.trim()
  if (!name) return
  emit('saveName', name)
  editing.value = false
}

function submitLoadout(silent = false) {
  const [p1, p2, p3] = presets.value
  emit('saveLoadout', {
    weaponId: current.value.weaponId, armorId: current.value.armorId,
    chestRigId: current.value.chestRigId, backpackId: current.value.backpackId,
    helmetId: current.value.helmetId, headsetId: current.value.headsetId,
    consumables: current.value.consumables,
    presetWeaponId: p1.weaponId, presetArmorId: p1.armorId,
    presetChestRigId: p1.chestRigId, presetBackpackId: p1.backpackId,
    presetHelmetId: p1.helmetId, presetHeadsetId: p1.headsetId,
    presetConsumables: p1.consumables, presetName: p1.name,
    preset2WeaponId: p2.weaponId, preset2ArmorId: p2.armorId,
    preset2ChestRigId: p2.chestRigId, preset2BackpackId: p2.backpackId,
    preset2HelmetId: p2.helmetId, preset2HeadsetId: p2.headsetId,
    preset2Consumables: p2.consumables, preset2Name: p2.name,
    preset3WeaponId: p3.weaponId, preset3ArmorId: p3.armorId,
    preset3ChestRigId: p3.chestRigId, preset3BackpackId: p3.backpackId,
    preset3HelmetId: p3.helmetId, preset3HeadsetId: p3.headsetId,
    preset3Consumables: p3.consumables, preset3Name: p3.name,
  }, silent)
}

// 实时装备变更自动保存（防抖，仅预设走显式保存按钮）
let saveTimer: ReturnType<typeof setTimeout> | undefined
watch(() => current.value, () => {
  if (syncing) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => submitLoadout(true), 600)
}, { deep: true })
</script>

<template>
  <section class="view-page character-view">
    <header class="character-identity">
      <div class="character-avatar"><el-icon><User /></el-icon><span class="status-dot" /></div>
      <div class="character-copy">
        <span class="eyebrow">玩家角色 / ID {{ player.id.toString().padStart(4, '0') }}</span>
        <div class="character-name-row">
          <template v-if="editing">
            <el-input v-model="nameDraft" maxlength="16" autofocus @keyup.enter="submitName" />
            <el-button :icon="Check" :loading="savingName" circle title="保存名称" @click="submitName" />
          </template>
          <template v-else>
            <h1>{{ player.name }}</h1>
            <el-button :icon="EditPen" circle title="修改名称" @click="editing = true" />
          </template>
        </div>
        <p>{{ player.desc }}</p>
      </div>
      <div class="character-state">
        <span>当前状态</span>
        <strong :class="player.injury && player.injury !== 'none' ? 'text-danger' : 'text-success'">
          {{ player.injury && player.injury !== 'none' ? player.injury : '状态正常' }}
        </strong>
      </div>
    </header>

    <div class="attribute-strip">
      <div v-for="attr in mainAttributes" :key="attr.label" class="attribute-cell">
        <span>{{ attr.code }}</span><strong>{{ attr.value }}</strong><small>{{ attr.label }}</small>
      </div>
    </div>

    <div class="character-grid">
      <section class="surface-panel skill-section">
        <div class="panel-heading"><div><span>SKILL</span><h2>生存技能</h2></div><small>训练值 / 100</small></div>
        <div class="progress-list">
          <div v-for="skill in skills" :key="skill[0]" class="progress-row">
            <span>{{ skill[0] }}</span><el-progress :percentage="skill[1]" :show-text="false" /><b>{{ skill[1] }}</b>
          </div>
        </div>
      </section>
      <section class="surface-panel skill-section">
        <div class="panel-heading"><div><span>PROF</span><h2>武器熟练度</h2></div><small>实战成长</small></div>
        <div class="progress-list">
          <div v-for="prof in proficiencies" :key="prof[0]" class="progress-row">
            <span>{{ prof[0] }}</span><el-progress :percentage="prof[1]" :show-text="false" /><b>{{ prof[1] }}</b>
          </div>
        </div>
      </section>
    </div>

    <section class="loadout-panel surface-panel">
      <div class="panel-heading"><div><span>LOADOUT</span><h2>装备配置</h2></div><small>当前装备不计入仓库容量</small></div>

      <div class="loadout-area">
        <div class="loadout-block">
          <div class="loadout-block__heading"><strong>当前装备</strong><span>点击栏位从仓库装备</span></div>
          <div class="current-spacer" />
          <div class="slot-grid">
            <div v-for="row in gridRows" :key="row" class="slot-row">
              <button v-for="s in slotDefs.filter((x) => x.row === row)" :key="s.key" type="button" class="slot-cell" @click="openPicker('current', s.key)">
                <span class="slot-label">{{ s.label }}</span>
                <strong :class="{ empty: !slotName('current', s.key) }">{{ slotName('current', s.key) || '空' }}</strong>
              </button>
            </div>
          </div>
          <div class="consumable-block">
            <span class="consumable-block__label">随身补给</span>
            <div class="slot-row">
              <button v-for="i in consumableSlotCount" :key="i" type="button" class="slot-cell" @click="openConsumablePicker('current', i - 1)">
                <span class="slot-label">补给{{ i }}</span>
                <strong :class="{ empty: !consumableAt(current, i - 1) }">{{ consumableAt(current, i - 1) || '空' }}</strong>
              </button>
            </div>
          </div>
        </div>

        <div class="loadout-block">
          <div class="loadout-block__heading">
            <strong>预设装备</strong>
            <el-select v-model="presetIndex" size="small" class="preset-switch">
              <el-option v-for="(p, idx) in presets" :key="idx" :label="`预设 ${idx + 1} · ${p.name || '未命名'}`" :value="idx" />
            </el-select>
          </div>
          <div class="preset-name-row">
            <el-input v-model="presets[presetIndex].name" maxlength="10" placeholder="预设名称" />
            <small>失能丢装后按此预设补购 · 仅限商人装备</small>
          </div>
          <div class="slot-grid">
            <div v-for="row in gridRows" :key="row" class="slot-row">
              <button v-for="s in slotDefs.filter((x) => x.row === row)" :key="s.key" type="button" class="slot-cell" @click="openPicker('preset', s.key)">
                <span class="slot-label">{{ s.label }}</span>
                <strong :class="{ empty: !slotName('preset', s.key) }">{{ slotName('preset', s.key) || '空' }}</strong>
              </button>
            </div>
          </div>
          <div class="consumable-block">
            <span class="consumable-block__label">预设补给</span>
            <div class="slot-row">
              <button v-for="i in consumableSlotCount" :key="i" type="button" class="slot-cell" @click="openConsumablePicker('preset', i - 1)">
                <span class="slot-label">补给{{ i }}</span>
                <strong :class="{ empty: !consumableAt(presets[presetIndex], i - 1) }">{{ consumableAt(presets[presetIndex], i - 1) || '空' }}</strong>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="carry-strip">
        <span>可携带格数 <b>{{ liveCapacity.usedSlots }} / {{ liveCapacity.totalSlots }}</b></span>
        <span>可携带负重 <b>{{ liveCapacity.usedWeight.toFixed(1) }} / {{ liveCapacity.totalWeight.toFixed(1) }} kg</b></span>
        <small>基础 {{ liveCapacity.baseWeight.toFixed(1) }}kg + 胸挂/背包加成 {{ liveCapacity.bonusWeight.toFixed(1) }}kg</small>
      </div>
      <div class="loadout-actions"><span>预设补购受现金限制，且不计入仓库容量</span><el-button type="primary" :loading="savingLoadout" @click="submitLoadout">保存装备配置</el-button></div>
    </section>

    <div class="trait-band"><span>角色特质</span><strong>{{ player.trait }}</strong><p>疲劳 {{ player.fatigue }} · 压力 {{ player.stress }}</p></div>

    <el-dialog v-model="pickerOpen" :title="pickerTitle" width="440px">
      <div class="slot-picker-list">
        <button type="button" class="slot-picker-item" @click="pickOption('')">
          <span class="slot-picker-item__name">不装备</span><small>清除该栏位</small>
        </button>
        <button v-for="item in pickerList" :key="item.id" type="button" class="slot-picker-item" @click="pickOption(item.id)">
          <span class="slot-picker-item__name">{{ item.name }}</span><small>{{ item.detail }}</small>
        </button>
      </div>
    </el-dialog>
  </section>
</template>
