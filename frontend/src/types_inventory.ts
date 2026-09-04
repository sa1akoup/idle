// 仓库、装备预设与 Session 类型：拆分高频业务数据，保持 types.ts 的统一导出入口。
import type { ActionStyle, LootSummary } from './types_common'

export type RecoveryMethod = 'inventory' | 'hideout' | 'merchant'

export interface RecoveryChoice {
  targetPercent: number
  primaryMethod: RecoveryMethod
  fallbackMethod: RecoveryMethod | 'none'
}

export interface RecoveryPolicy {
  hp: RecoveryChoice
  energy: RecoveryChoice
  hydration: RecoveryChoice
  merchantEnable: boolean
}

// AmmoCell 是随身携带弹药的一个槽位：弹药 ID + 携弹发数（1-60）。
export interface AmmoCell {
  ammoId: string
  rounds: number
}

export interface InventoryItem {
  id: number
  itemId: string
  name: string
  kind: 'currency' | 'material' | 'loot' | 'weapon' | 'armor' | 'ammo' | 'consumable' | 'chestrig' | 'backpack' | 'helmet' | 'headset' | 'keycase' | 'secure'
  category: string
  quantity: number
  price: number
  weight: number
  slots: number
  raidExtract: boolean
  merchantCategory: string
  repRequirement: number
  purposes?: string[]
  updatedAt: string
}

export interface Merchant {
  id: string
  name: string
  category: string
  reputation: number
  desc: string
  open: boolean
  sortOrder: number
}

export interface MerchantBarterCost {
  itemId: string
  name: string
  need: number
  have: number
}

export interface MerchantCatalogItem {
  id: string
  name: string
  kind: string
  buyable: boolean
  category: string
  detail: string
  basePrice: number
  price: number
  sellPrice: number
  weight: number
  slots: number
  repRequirement: number
  hpRecovery: number
  energyRecovery: number
  hydrationRecovery: number
  repairValue: number
  fuelSeconds: number
  maxDurability: number
  instanceRequired: boolean
  stock?: number
  barterCosts?: MerchantBarterCost[]
  barterLocked?: boolean
  barterLockReason?: string
}

export interface MerchantCatalogResponse {
  items: MerchantCatalogItem[]
  nextRefreshAt?: string
  acceptsAny: boolean
  playerSellRate: number
}

export interface KeySlotView {
  slotIndex: number
  instanceId: number
  itemId: string
  name: string
  currentDurability: number
  maxDurability: number
}

export interface PlayerLoadout {
  id: number
  characterId: number
  weaponId: string
  armorId: string
  chestRigId: string
  backpackId: string
  helmetId: string
  headsetId: string
  keyCaseId?: string
  keyCaseSlots?: number
  keys?: KeySlotView[]
  secureContainerId?: string
  consumables: string[]
  carriedAmmo?: AmmoCell[]
  presetWeaponId: string
  presetArmorId: string
  presetChestRigId: string
  presetBackpackId: string
  presetHelmetId: string
  presetHeadsetId: string
  presetConsumables: string[]
  presetAmmoId: string
  presetAmmoRounds: number
  presetName: string
  preset2WeaponId: string
  preset2ArmorId: string
  preset2ChestRigId: string
  preset2BackpackId: string
  preset2HelmetId: string
  preset2HeadsetId: string
  preset2Consumables: string[]
  preset2AmmoId: string
  preset2AmmoRounds: number
  preset2Name: string
  preset3WeaponId: string
  preset3ArmorId: string
  preset3ChestRigId: string
  preset3BackpackId: string
  preset3HelmetId: string
  preset3HeadsetId: string
  preset3Consumables: string[]
  preset3AmmoId: string
  preset3AmmoRounds: number
  preset3Name: string
  updatedAt: string
}

export interface KeyCase {
  id: string
  name: string
  keySlots: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface SecureContainer {
  id: string
  name: string
  innerSlots: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
  unlockQuestId?: string
}

export interface SaveLoadoutRequest {
  weaponId: string
  armorId: string
  chestRigId: string
  backpackId: string
  helmetId: string
  headsetId: string
  keyCaseId?: string
  secureContainerId?: string
  keyInstanceIds?: number[]
  consumables: string[]
  carriedAmmo?: AmmoCell[]
  presetWeaponId: string
  presetArmorId: string
  presetChestRigId: string
  presetBackpackId: string
  presetHelmetId: string
  presetHeadsetId: string
  presetConsumables: string[]
  presetAmmoId: string
  presetAmmoRounds: number
  presetName: string
  preset2WeaponId: string
  preset2ArmorId: string
  preset2ChestRigId: string
  preset2BackpackId: string
  preset2HelmetId: string
  preset2HeadsetId: string
  preset2Consumables: string[]
  preset2AmmoId: string
  preset2AmmoRounds: number
  preset2Name: string
  preset3WeaponId: string
  preset3ArmorId: string
  preset3ChestRigId: string
  preset3BackpackId: string
  preset3HelmetId: string
  preset3HeadsetId: string
  preset3Consumables: string[]
  preset3AmmoId: string
  preset3AmmoRounds: number
  preset3Name: string
}

// 预设装备序号对应的 loadout 字段组合
export interface PresetSlot {
  name: string
  weaponId: string
  armorId: string
  consumables: string[]
  ammoId: string
  ammoRounds: number
}

export function presetOf(loadout: PlayerLoadout, index: number): PresetSlot {
  switch (index) {
    case 2:
      return { name: loadout.preset2Name ?? '', weaponId: loadout.preset2WeaponId, armorId: loadout.preset2ArmorId, consumables: loadout.preset2Consumables, ammoId: loadout.preset2AmmoId, ammoRounds: loadout.preset2AmmoRounds }
    case 3:
      return { name: loadout.preset3Name ?? '', weaponId: loadout.preset3WeaponId, armorId: loadout.preset3ArmorId, consumables: loadout.preset3Consumables, ammoId: loadout.preset3AmmoId, ammoRounds: loadout.preset3AmmoRounds }
    default:
      return { name: loadout.presetName ?? '', weaponId: loadout.presetWeaponId, armorId: loadout.presetArmorId, consumables: loadout.presetConsumables, ammoId: loadout.presetAmmoId, ammoRounds: loadout.presetAmmoRounds }
  }
}

export interface Session {
  id: number
  characterId: number
  mapId: string
  style: ActionStyle
  recoveryPreset: number
  weaponId: string
  armorId: string
  ammoId: string
  ammoRounds: number
  consumables: string
  status: 'running' | 'success' | 'incapacitated' | 'failed'
  seed: number
  startTime: string
  endTime: string | null
  offlineLimitMin: number
  elapsedMin: number
  offlineLimitSec: number
  elapsedSec: number
  currentRunStartedAt: string | null
  nextRunAt: string | null
  lastProcessedAt: string | null
  heartbeatAt: string | null
  engineVersion: string
  scenarioHash: string
  pendingRunIndex: number
  pendingRunHash: string
  totalRuns: number
  createdAt: string
}

export type SessionEventType =
  | 'run_started'
  | 'route_planned'
  | 'node_entered'
  | 'node_move_started'
  | 'event_triggered'
  | 'evacuation_started'
  | 'container_search_started'
  | 'loot_found'
  | 'loot_collected'
  | 'container_search_finished'
  | 'battle_started'
  | 'battle_attack'
  | 'battle_round'
  | 'battle_escape'
  | 'battle_finished'
  | 'extraction_approach'
  | 'extraction_point_reached'
  | 'extraction_completed'
  | 'loot_extracted'
  | 'loot_stored'
  | 'loot_secured'
  | 'loot_overflow'
  | 'ammo_refilled'
  | 'run_settled'
  | 'session_finished'
  | 'session_failed'

export interface SessionEvent {
  id: number
  sessionId: number
  runIndex: number
  sequence: number
  eventType: SessionEventType
  offsetSec: number
  availableAt: string
  nodeId: string
  subjectId: string
  payload: Record<string, unknown>
  createdAt: string
}

export interface SessionRunRaw {
  id: number
  sessionId: number
  runIndex: number
  result: 'success' | 'partial' | 'emergency' | 'captured' | 'incapacitated'
  durationMin: number
  durationSec: number
  heat: number
  ammoUsed: number
  startHp: number
  endHp: number
  startEnergy: number
  endEnergy: number
  startHydration: number
  endHydration: number
  loot: string
  storedLoot: string
  overflowLoot: string
  consumedItems: string
  report: string
  createdAt: string
}

export interface SessionRun extends Omit<SessionRunRaw, 'loot' | 'report'> {
  loot: LootSummary[]
  report: string[]
}

export interface SessionDetail {
  session: Session
  runs: SessionRunRaw[]
}

export interface StartSessionRequest {
  mapId: string
  style: ActionStyle
  recoveryPreset: number
  recoveryPolicy: RecoveryPolicy
}
