// 角色装备逻辑：维护当前装备、三套预设、选项弹窗和携行容量计算。
import { computed, ref, watch } from 'vue'
import { emptySet, fromLoadout, gridRows, presetFromLoadout, slotDefs, type EquipSet, type SlotKey } from './characterLoadoutHelpers'
import type {
  Ammo,
  Armor,
  Backpack,
  ChestRig,
  Consumable,
  Headset,
  Helmet,
  InventoryItem,
  Player,
  PlayerLoadout,
  SaveLoadoutRequest,
  Weapon,
} from '../types'

export interface CharacterProps {
  player: Player
  loadout: PlayerLoadout
  inventory: InventoryItem[]
  weapons: Weapon[]
  ammos: Ammo[]
  armors: Armor[]
  consumables: Consumable[]
  chestRigs: ChestRig[]
  backpacks: Backpack[]
  helmets: Helmet[]
  headsets: Headset[]
  savingName: boolean
  savingLoadout: boolean
}

export type CharacterEmit = {
  (event: 'saveName', name: string): void
  (event: 'saveLoadout', request: SaveLoadoutRequest, silent?: boolean): void
}

export function useCharacterLoadout(props: CharacterProps, emit: CharacterEmit) {

  
  const editing = ref(false)
  const nameDraft = ref(props.player.name)
  
  
  const current = ref<EquipSet>(emptySet())
  const presets = ref<EquipSet[]>([emptySet('标准突击'), emptySet('轻装渗透'), emptySet('重装攻坚')])
  const presetIndex = ref(0)
  
  
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
      weaponId: props.weapons.map((i) => ({ name: i.name, detail: `伤害 ${i.damage} · ${i.ammoPerRound > 0 ? `口径 ${i.caliberId}` : `近战穿透 ${i.penetration}`} · ${i.weight}kg${repTag(i.repRequirement)}` })),
      armorId: props.armors.map((i) => ({ name: i.name, detail: `A${i.protectionLevel} · 覆盖 ${i.coverage}% · ${i.weight}kg${repTag(i.repRequirement)}` })),
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
      if (pickerContext.value === 'preset' && pickerKey.value === 'weaponId') normalizePresetAmmo(set)
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
  
  const activePresetWeapon = computed(() => props.weapons.find((item) => item.id === presets.value[presetIndex.value].weaponId))
  const activePresetAmmoOptions = computed(() => {
    const caliberId = activePresetWeapon.value?.caliberId
    return caliberId ? props.ammos.filter((item) => item.caliberId === caliberId).sort((left, right) => left.level - right.level) : []
  })
  
  function normalizePresetAmmo(set: EquipSet) {
    const weapon = props.weapons.find((item) => item.id === set.weaponId)
    if (!weapon || weapon.ammoPerRound <= 0) {
      set.ammoId = ''
      set.ammoRounds = 0
      return
    }
    const compatible = props.ammos.filter((item) => item.caliberId === weapon.caliberId)
    if (!compatible.some((item) => item.id === set.ammoId)) set.ammoId = compatible.find((item) => item.level === 4)?.id ?? compatible[0]?.id ?? ''
    if (set.ammoRounds < weapon.ammoPerRound) set.ammoRounds = Math.max(weapon.ammoPerRound, 60)
  }
  
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
      presetConsumables: p1.consumables, presetAmmoId: p1.ammoId, presetAmmoRounds: p1.ammoRounds, presetName: p1.name,
      preset2WeaponId: p2.weaponId, preset2ArmorId: p2.armorId,
      preset2ChestRigId: p2.chestRigId, preset2BackpackId: p2.backpackId,
      preset2HelmetId: p2.helmetId, preset2HeadsetId: p2.headsetId,
      preset2Consumables: p2.consumables, preset2AmmoId: p2.ammoId, preset2AmmoRounds: p2.ammoRounds, preset2Name: p2.name,
      preset3WeaponId: p3.weaponId, preset3ArmorId: p3.armorId,
      preset3ChestRigId: p3.chestRigId, preset3BackpackId: p3.backpackId,
      preset3HelmetId: p3.helmetId, preset3HeadsetId: p3.headsetId,
      preset3Consumables: p3.consumables, preset3AmmoId: p3.ammoId, preset3AmmoRounds: p3.ammoRounds, preset3Name: p3.name,
    }, silent)
  }
  
  // 实时装备变更自动保存（防抖，仅预设走显式保存按钮）
  let saveTimer: ReturnType<typeof setTimeout> | undefined
  watch(() => current.value, () => {
    if (syncing) return
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => submitLoadout(true), 600)
  }, { deep: true })

  return {
    editing, nameDraft, current, presets, presetIndex, gridRows, slotDefs,
    mainAttributes, skills, proficiencies, openPicker, openConsumablePicker,
    slotName, consumableSlotCount, consumableAt, activePresetWeapon, activePresetAmmoOptions,
    repTag, liveCapacity, pickerOpen, pickerTitle, pickerList, pickOption,
    submitName, submitLoadout,
  }
}
