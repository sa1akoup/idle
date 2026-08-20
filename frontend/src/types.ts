// 游戏领域类型：集中约束玩家、装备、地图与行动报告的数据结构。
export type NavKey =
  | 'explore'
  | 'map'
  | 'character'
  | 'inventory'
  | 'merchant'
  | 'hideout'
  | 'logs'

export type ActionStyle = 'balanced' | 'stealth' | 'aggressive' | 'greedy'

export interface Player {
  id: number
  name: string
  desc: string
  strength: number
  agility: number
  intellect: number
  charisma: number
  stealth: number
  perception: number
  negotiation: number
  luck: number
  survival: number
  resist: number
  engineering: number
  medical: number
  meleeProf: number
  pistolProf: number
  smgProf: number
  shotgunProf: number
  rifleProf: number
  sniperProf: number
  trait: string
  fatigue: number
  stress: number
  injury: 'none' | 'light' | 'heavy' | 'lethal' | ''
  injuryUntil: string | null
  createdAt: string
}

export interface Weapon {
  id: string
  name: string
  category: string
  hit: number
  damage: number
  penetration: number
  suppress: number
  ready: number
  ammoPerRound: number
  noise: number
  reliability: number
  closeMod: number
  midMod: number
  farMod: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface Armor {
  id: string
  name: string
  type: 'light' | 'heavy'
  protect: number
  coverage: number
  mobility: number
  initiative: number
  conceal: number
  antiSuppress: number
  escape: number
  maxDurability: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface ChestRig {
  id: string
  name: string
  addSlots: number
  addWeight: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface Backpack {
  id: string
  name: string
  addSlots: number
  addWeight: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface Helmet {
  id: string
  name: string
  protect: number
  coverage: number
  mobility: number
  initiative: number
  conceal: number
  antiSuppress: number
  escape: number
  maxDurability: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface Headset {
  id: string
  name: string
  hearingLevel: number
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface CarryCapacity {
  baseSlots: number
  bonusSlots: number
  totalSlots: number
  baseWeight: number
  bonusWeight: number
  totalWeight: number
  usedSlots: number
  usedWeight: number
}

export interface StorageCapacity {
  capacity: number
  used: number
}

export interface ArmorInstance {
  id: number
  armorId: string
  maxDurability: number
  curDurability: number
  repairCount: number
  status: 'normal' | 'repairing' | 'broken'
}

export interface Consumable {
  id: string
  name: string
  desc: string
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface LootItem {
  id: string
  name: string
  category: 'tool' | 'material' | 'electronics' | 'info' | 'medical' | 'food' | 'valuable' | 'fuel' | 'weaponpart'
  desc: string
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
}

export interface LootSummary {
  itemId: string
  name: string
  category: string
  quantity: number
  containerId: string
  source: string
}

export interface GameMap {
  id: string
  name: string
  desc: string
  startNodeId: string
  extractionNodeId: string
  tags: string[]
}

export interface NodeContainerView {
  id: string
  name: string
  pool: string
  tags: string[]
  valueTier: number
  weight: number
  searchRisk: number
  searchTime: number
  count: number
}

export interface MapNode {
  id: string
  mapId: string
  name: string
  routeOrder: number
  exploreTime: number
  distance: 'close' | 'mid' | 'far'
  enemyId: string
  encounterRole: string
  containerSlots: number
  valueTier: number
  connections: string
  tags: string[]
  containers: NodeContainerView[]
}

export interface Enemy {
  id: string
  name: string
  hp: number
  stressThreshold: number
  perception: number
  stealth: number
  agility: number
  weaponId: string
  armorId: string
  evasion: number
  mobility: number
  suppress: number
  backpackContainerId: string
}

export interface InventoryItem {
  id: number
  itemId: string
  name: string
  kind: 'currency' | 'material' | 'loot' | 'weapon' | 'armor' | 'consumable' | 'chestrig' | 'backpack' | 'helmet' | 'headset'
  category: string
  quantity: number
  price: number
  weight: number
  slots: number
  raidExtract: boolean
  merchantCategory: string
  repRequirement: number
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

export interface MerchantCatalogItem {
  id: string
  name: string
  kind: string
  detail: string
  basePrice: number
  price: number
  sellPrice: number
  weight: number
  slots: number
  repRequirement: number
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
  consumables: string[]
  presetWeaponId: string
  presetArmorId: string
  presetChestRigId: string
  presetBackpackId: string
  presetHelmetId: string
  presetHeadsetId: string
  presetConsumables: string[]
  presetName: string
  preset2WeaponId: string
  preset2ArmorId: string
  preset2ChestRigId: string
  preset2BackpackId: string
  preset2HelmetId: string
  preset2HeadsetId: string
  preset2Consumables: string[]
  preset2Name: string
  preset3WeaponId: string
  preset3ArmorId: string
  preset3ChestRigId: string
  preset3BackpackId: string
  preset3HelmetId: string
  preset3HeadsetId: string
  preset3Consumables: string[]
  preset3Name: string
  updatedAt: string
}

export interface SaveLoadoutRequest {
  weaponId: string
  armorId: string
  chestRigId: string
  backpackId: string
  helmetId: string
  headsetId: string
  consumables: string[]
  presetWeaponId: string
  presetArmorId: string
  presetChestRigId: string
  presetBackpackId: string
  presetHelmetId: string
  presetHeadsetId: string
  presetConsumables: string[]
  presetName: string
  preset2WeaponId: string
  preset2ArmorId: string
  preset2ChestRigId: string
  preset2BackpackId: string
  preset2HelmetId: string
  preset2HeadsetId: string
  preset2Consumables: string[]
  preset2Name: string
  preset3WeaponId: string
  preset3ArmorId: string
  preset3ChestRigId: string
  preset3BackpackId: string
  preset3HelmetId: string
  preset3HeadsetId: string
  preset3Consumables: string[]
  preset3Name: string
}

// 预设装备序号对应的 loadout 字段组合
export interface PresetSlot {
  weaponId: string
  armorId: string
  consumables: string[]
}

export function presetOf(loadout: PlayerLoadout, index: number): PresetSlot {
  switch (index) {
    case 2:
      return { weaponId: loadout.preset2WeaponId, armorId: loadout.preset2ArmorId, consumables: loadout.preset2Consumables }
    case 3:
      return { weaponId: loadout.preset3WeaponId, armorId: loadout.preset3ArmorId, consumables: loadout.preset3Consumables }
    default:
      return { weaponId: loadout.presetWeaponId, armorId: loadout.presetArmorId, consumables: loadout.presetConsumables }
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
  consumables: string
  status: 'running' | 'waiting_injury' | 'finished' | 'aborted' | 'failed'
  seed: number
  startTime: string
  endTime: string | null
  offlineLimitMin: number
  totalRuns: number
  createdAt: string
}

export interface SessionRunRaw {
  id: number
  sessionId: number
  runIndex: number
  result: 'success' | 'partial' | 'emergency' | 'captured' | 'incapacitated'
  durationMin: number
  heat: number
  ammoUsed: number
  injury: string
  loot: string
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

export type PreviewResult = Record<string, string>

export interface StartSessionRequest {
  mapId: string
  style: ActionStyle
  recoveryPreset: number
}
