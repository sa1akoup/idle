// 角色装备逻辑：维护当前装备、随身弹药、三套预设、选项弹窗和携行容量计算。
import { computed, ref, watch } from 'vue'
import {
  ammoCellsFromLoadout, ammoSlotCount, ammoSlotMaxRounds, emptySet, fromLoadout, gridRows,
  presetFromLoadout, slotDefs, type EquipSet, type SlotKey,
} from './characterLoadoutHelpers'
import type {
  Ammo,
  Armor,
  Backpack,
  ChestRig,
  Consumable,
  Headset,
  Helmet,
  InventoryItem,
  ItemInstance,
  Merchant,
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
  merchants: Merchant[]
  armors: Armor[]
  consumables: Consumable[]
  itemInstances: ItemInstance[]
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
  (event: 'dirtyChange', dirty: boolean): void
}

// equipSetContentKey 提取装备或预设的可编辑字段，供 dirty 状态和轮询同步判断。
function equipSetContentKey(set: EquipSet, includePresetFields = false): unknown[] {
  const content: unknown[] = [
    set.weaponId, set.armorId, set.chestRigId, set.backpackId, set.helmetId, set.headsetId, set.consumables,
    set.ammo.map((cell) => [cell.ammoId, cell.rounds]),
  ]
  if (includePresetFields) content.push(set.name, set.ammoId, set.ammoRounds)
  return content
}

// loadoutContentKey 生成当前装备与三套预设的稳定指纹，避免只保护当前装备而漏掉预设编辑。
function loadoutContentKey(current: EquipSet, presets: EquipSet[] = []): string {
  return JSON.stringify([
    equipSetContentKey(current),
    ...presets.map((preset) => equipSetContentKey(preset, true)),
  ])
}

export function useCharacterLoadout(props: CharacterProps, emit: CharacterEmit) {


  const editing = ref(false)
  const nameDraft = ref(props.player.name)
  
  
  const current = ref<EquipSet>(emptySet())
  const presets = ref<EquipSet[]>([emptySet('标准突击'), emptySet('轻装渗透'), emptySet('重装攻坚')])
  const presetIndex = ref(0)
  
  
  // loadout 同步：记录最近一次同步内容的指纹，供未保存状态判定；
  // 有未保存修改时跳过覆盖 current，防止后台轮询刷新静默吞掉正在编辑的装备。
  let lastSyncedKey = ''
  let initialSyncDone = false
  watch(() => props.loadout, (l) => {
    const next = fromLoadout(l)
    const nextPresets = [presetFromLoadout(l, 0), presetFromLoadout(l, 1), presetFromLoadout(l, 2)]
    const nextKey = loadoutContentKey(next, nextPresets)
    const currentKey = loadoutContentKey(current.value, presets.value)
    if (!initialSyncDone) {
      lastSyncedKey = nextKey
      current.value = next
      initialSyncDone = true
      presets.value = nextPresets
    } else if (nextKey === currentKey || currentKey === lastSyncedKey) {
      // 刷新内容与当前一致（保存成功后的回刷）或当前无未保存修改：正常应用刷新
      lastSyncedKey = nextKey
      current.value = next
      presets.value = nextPresets
    }
    // 任一装备区域有未保存修改时，保留当前装备与预设，避免轮询覆盖用户编辑。
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
  const usableInstanceItemIds = computed(() => new Set(props.itemInstances
    .filter((instance) => instance.locationType === 'inventory' && instance.status === 'normal' && instance.currentDurability > 0)
    .map((instance) => instance.itemId)))
  
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
  const pickerKind = ref<'equip' | 'consumable' | 'ammo'>('equip')
  const pickerContext = ref<'current' | 'preset'>('current')
  const pickerKey = ref<SlotKey>('weaponId')
  const pickerConsumableIndex = ref(0)
  const pickerAmmoIndex = ref(0)
  
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
  function openAmmoPicker(context: 'current', index: number) {
    pickerKind.value = 'ammo'
    pickerContext.value = context
    pickerAmmoIndex.value = index
    pickerOpen.value = true
  }
  function pickOption(id: string) {
    const set = activeSet(pickerContext.value)
    if (pickerKind.value === 'consumable') {
      const arr = padConsumables(set.consumables)
      arr[pickerConsumableIndex.value] = id
      set.consumables = arr.filter(Boolean)
    } else if (pickerKind.value === 'ammo') {
      const cells = ammoCellsFromLoadout(set.ammo)
      const cell = cells[pickerAmmoIndex.value] ?? { ammoId: '', rounds: 0 }
      cell.ammoId = id
      if (id === '') cell.rounds = 0
      if (cell.rounds < 1) cell.rounds = ammoDefaultRounds(id)
      cells[pickerAmmoIndex.value] = cell
      set.ammo = cells
    } else {
      set[pickerKey.value] = id
      if (pickerContext.value === 'preset' && pickerKey.value === 'weaponId') {
        normalizePresetAmmo(set)
      } else if (pickerContext.value === 'current' && pickerKey.value === 'weaponId') {
        normalizeCurrentAmmo(set)
      }
    }
    pickerOpen.value = false
  }

  // 当前武器切换后清理异口径或近战遗留弹药，避免保存成功但启动时才发现口径冲突。
  function normalizeCurrentAmmo(set: EquipSet) {
    const weapon = props.weapons.find((item) => item.id === set.weaponId)
    if (!weapon || weapon.ammoPerRound <= 0) {
      set.ammo = ammoCellsFromLoadout([])
      return
    }
    set.ammo = ammoCellsFromLoadout(set.ammo).map((cell) => {
      const ammo = props.ammos.find((item) => item.id === cell.ammoId)
      return ammo?.caliberId === weapon.caliberId ? cell : { ammoId: '', rounds: 0 }
    })
  }
  // 选择弹药后给一个可开火的默认发数（按当前武器单轮消耗取整，封顶单格 60 发）。
  function ammoDefaultRounds(ammoId: string): number {
    const ammo = props.ammos.find((item) => item.id === ammoId)
    const weapon = props.weapons.find((item) => item.id === current.value.weaponId)
    if (!ammo || !weapon || weapon.ammoPerRound <= 0 || ammo.caliberId !== weapon.caliberId) return ammo ? 60 : 0
    const base = Math.ceil(60 / weapon.ammoPerRound) * weapon.ammoPerRound
    return Math.min(60, Math.max(base, weapon.ammoPerRound))
  }
  // 随身弹药备选：仓库中与当前武器口径匹配的实弹；未装备枪械时提供全部库存实弹。
  const ammoPickerOptions = computed(() => {
    const weapon = props.weapons.find((item) => item.id === current.value.weaponId)
    return props.ammos
      .filter((ammo) => {
        if (weapon && weapon.ammoPerRound > 0 && ammo.caliberId !== weapon.caliberId) return false
        return ammoInventoryQuantityOf(ammo.id) > 0
      })
      .sort((left, right) => right.level - left.level)
      .map((ammo) => ({
        id: ammo.id,
        name: ammo.name,
        detail: `${ammo.caliberId} · 库存 ${ammoInventoryQuantityOf(ammo.id)} 发 · 每格上限 ${ammoSlotMaxRounds} 发`,
      }))
  })
  function ammoInventoryQuantityOf(ammoId: string): number {
    return props.inventory
      .filter((item) => item.kind === 'ammo' && item.itemId === ammoId)
      .reduce((total, item) => total + item.quantity, 0)
  }
  // 已预留到 Session 的弹药不再计入仓库库存，但当前配置仍需保持可编辑且不被钳成非法值。
  function ammoCellMaxRounds(ammoId: string, configuredRounds = 0): number {
    if (!ammoId) return 0
    const configuredMax = Math.min(Math.max(configuredRounds, 0), ammoSlotMaxRounds)
    return Math.max(configuredMax, Math.min(ammoSlotMaxRounds, ammoInventoryQuantityOf(ammoId)))
  }
  function ammoNameAt(set: EquipSet, index: number): string {
    const cell = (set.ammo ?? [])[index]
    if (!cell?.ammoId) return ''
    return props.ammos.find((item) => item.id === cell.ammoId)?.name ?? cell.ammoId
  }
  function ammoAt(set: EquipSet, index: number): { ammoId: string; rounds: number } {
    const cell = (set.ammo ?? [])[index]
    return cell ?? { ammoId: '', rounds: 0 }
  }
  const pickerList = computed(() => {
    if (pickerKind.value === 'consumable') {
      return props.consumables
        .filter((c) => c.usableInSession && (pickerContext.value === 'preset' || ownedItemIds.value.has(c.id) || usableInstanceItemIds.value.has(c.id)))
        .map((c) => ({ id: c.id, name: c.name, detail: `${c.desc} · ${c.weight}kg` }))
    }
    if (pickerKind.value === 'ammo') return ammoPickerOptions.value
    return pickerContext.value === 'current' ? currentOptions(pickerKey.value) : slotOptions(pickerKey.value)
  })
  const pickerTitle = computed(() => {
    if (pickerKind.value === 'consumable') {
      return `${pickerContext.value === 'current' ? '随身' : '预设'}补给 · ${pickerContext.value === 'current' ? '仓库道具' : '商人道具'}`
    }
    if (pickerKind.value === 'ammo') return '随身弹药 · 仓库实弹'
    return `${slotDefs.find((s) => s.key === pickerKey.value)?.label} · ${pickerContext.value === 'current' ? '仓库装备' : '商人装备'}`
  })
  
  // 武器商人当前好感度
  const weaponMerchantReputation = computed(() =>
    props.merchants.find((item) => item.category === 'weapon')?.reputation ?? 0
  )
  // 商人可售弹药：武器商人开放、同口径、等级 ≤ 4、好感度达标（与后端 scenario_snapshot.go 的 Available 规则一致）
  function merchantAmmoOptions(caliberId: string | undefined): Ammo[] {
    if (!caliberId) return []
    const rep = weaponMerchantReputation.value
    return props.ammos
      .filter((item) => item.caliberId === caliberId && item.level <= 4 && item.repRequirement <= rep)
      .sort((left, right) => left.level - right.level)
  }

  const activePresetWeapon = computed(() => props.weapons.find((item) => item.id === presets.value[presetIndex.value].weaponId))
  const activePresetAmmoOptions = computed(() => merchantAmmoOptions(activePresetWeapon.value?.caliberId))

  function normalizePresetAmmo(set: EquipSet) {
    const weapon = props.weapons.find((item) => item.id === set.weaponId)
    if (!weapon || weapon.ammoPerRound <= 0) {
      set.ammoId = ''
      set.ammoRounds = 0
      return
    }
    // 默认取武器商人可购买的最低等级弹药（无好感度门槛），携带 30 发（不低于单次消耗）。
    const compatible = merchantAmmoOptions(weapon.caliberId)
    const defaultAmmo = compatible[0] // merchantAmmoOptions 已按等级升序排列
    if (!compatible.some((item) => item.id === set.ammoId)) set.ammoId = defaultAmmo?.id ?? ''
    if (set.ammoRounds < weapon.ammoPerRound) set.ammoRounds = Math.max(weapon.ammoPerRound, 30)
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
    const baseSlots = 5
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
  
    let usedWeight = items.reduce((s, i) => s + i.weight, 0)
    let usedSlots = items.reduce((s, i) => s + i.slots, 0)
    // 随身弹药槽参与容量计算：每格按整组折算格数与 0.5kg/组负重（与后端公式一致）。
    for (const cell of current.value.ammo ?? []) {
      if (!cell.ammoId || cell.rounds <= 0) continue
      const ammo = props.ammos.find((item) => item.id === cell.ammoId)
      if (!ammo || ammo.roundsPerSlot <= 0) continue
      const groups = Math.ceil(cell.rounds / ammo.roundsPerSlot)
      usedSlots += groups
      usedWeight += groups * 0.5
    }
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
      carriedAmmo: (current.value.ammo ?? []).map((cell) => ({ ammoId: cell.ammoId, rounds: cell.rounds })),
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
  
  // 未保存的当前装备调整：与最近一次后端同步的内容不一致即为有未保存修改，
  // 用于离开角色页前的提示；显式点击"保存装备配置"后由 loadout 回刷自动清零。
  const hasUnsavedChanges = computed(() => loadoutContentKey(current.value, presets.value) !== lastSyncedKey)

  return {
    editing, nameDraft, current, presets, presetIndex, gridRows, slotDefs,
    mainAttributes, skills, proficiencies, openPicker, openConsumablePicker, openAmmoPicker,
    slotName, consumableSlotCount, consumableAt, ammoSlotCount, ammoSlotMaxRounds, ammoAt, ammoNameAt, ammoCellMaxRounds,
    activePresetWeapon, activePresetAmmoOptions, repTag, liveCapacity, pickerOpen, pickerTitle, pickerList, pickOption,
    submitName, submitLoadout, hasUnsavedChanges,
  }
}
