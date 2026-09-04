// 游戏领域类型：集中约束玩家、装备、地图与行动报告的数据结构。
export type NavKey =
  | 'explore'
  | 'live'
  | 'map'
  | 'character'
  | 'inventory'
  | 'merchant'
  | 'hideout'
  | 'logs'

export interface Quest {
  id: string
  merchantId: string
  merchantName: string
  name: string
  description: string
  status: 'locked' | 'available' | 'active' | 'completed'
  objectiveType: 'extract_item' | 'defeat_kind' | 'visit_node' | 'style_extract' | 'survive_runs'
  targetId: string
  targetLabel: string
  required: number
  current: number
  canAccept: boolean
  canTurnIn: boolean
  rewardCash: number
  rewardRep: number
  prerequisiteId: string
}

export interface User {
	id: number
	username: string
	email?: string | null
	status: string
	createdAt: string
	updatedAt: string
}

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
  hp: number
  energy: number
  hydration: number
  needsUpdatedAt: string
  stress: number
  createdAt: string
  hpMax: number
  energyMax: number
  hydrationMax: number
  stressMax: number
  recoveryPerHour: {
    hp: number
    energy: number
    hydration: number
    stress: number
  }
}

export interface Weapon {
  id: string
  name: string
  category: string
  caliberId: string
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

export interface Ammo {
  id: string
  name: string
  caliberId: string
  level: number
  fleshDamageMultiplier: number
  armorDamageMultiplier: number
  price: number
  roundsPerSlot: number
  merchantCategory: string
  repRequirement: number
}

export interface Armor {
  id: string
  name: string
  type: 'light' | 'heavy'
  protect: number
  protectionLevel: number
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

export interface StorageCapacity {
  capacity: number
  used: number
}

export interface HideoutUpgrade {
  level: number
  originalCost: number
  originalCurrency: string
  originalSeconds: number
  cost: number
  durationSec: number
  materialId: string
  materialName: string
  materialQuantity: number
  effectsJson: string
  requirements: HideoutRequirement[]
  canStart: boolean
}

export interface HideoutRequirement {
  requirementType: string
  referenceId: string
  label: string
  quantity: number
  requiredValue: number
  currentValue: number
  satisfied: boolean
}

export interface HideoutFacility {
  id: string
  name: string
  category: string
  description: string
  iconKey: string
  level: number
  maxLevel: number
  state: 'ready' | 'upgrading'
  storageBonus: number
  recoverySpeedPercent: number
  repairSpeedPercent: number
  intelBonusPercent: number
  hpRecoveryPerHour: number
  energyRecoveryPerHour: number
  hydrationRecoveryPerHour: number
  repairKitDiscountPercent: number
  fuelConsumptionReductionPercent: number
  physicalSkillGrowthPercent: number
  stressRecoveryPerHour: number
  fuelSlotCount: number
  requiresPower: boolean
  effectsJson: string
  runtime: 'active' | 'planned'
  nextUpgrade: HideoutUpgrade | null
}

export interface HideoutJob {
  id: number
  facilityId: string
  jobType: 'upgrade' | 'repair' | 'craft' | 'training' | 'scav_case'
  targetLevel: number
  targetRef: string
  armorInstanceId: number | null
  startedAt: string
  completeAt: string
  status: 'running' | 'completed'
}

export interface HideoutBonuses {
  storageBonus: number
  recoverySpeedPercent: number
  repairSpeedPercent: number
  intelBonusPercent: number
  hpRecoveryPerHour: number
  energyRecoveryPerHour: number
  hydrationRecoveryPerHour: number
  repairKitDiscountPercent: number
  fuelConsumptionReductionPercent: number
  physicalSkillGrowthPercent: number
  stressRecoveryPerHour: number
}

export interface GeneratorFuel {
  instanceId: number
  itemId: string
  currentDurability: number
  maxDurability: number
  fuelSeconds: number
}

export interface GeneratorView {
  enabled: boolean
  fuelSlots: number
  fuelRemainingSeconds: number
  fuelConsumptionFactor: number
  updatedAt: string
  fuels: GeneratorFuel[]
}

export interface HideoutSnapshot {
  facilities: HideoutFacility[]
  jobs: HideoutJob[]
  bonuses: HideoutBonuses
  storageCapacity: StorageCapacity
  repairCost: number
  generator: GeneratorView | null
}

export interface CraftingMaterial {
  itemId: string
  name: string
  need: number
  have: number
  satisfied: boolean
}

export interface CraftingOutput {
  itemId: string
  name: string
  kind: string
  quantity: number
  instanceRequired: boolean
}

export interface CraftingRecipe {
  id: string
  name: string
  requiredLevel: number
  craftSeconds: number
  craftMinutes: number
  output: CraftingOutput
  inputs: CraftingMaterial[]
  facilityId: string
  facilityName: string
  facilityLevel: number
  workbenchLevel: number
  workbenchBusy: boolean
  canStart: boolean
  reason?: string
}

export interface ArmorInstance {
  id: number
  armorId: string
  maxDurability: number
  curDurability: number
  repairCount: number
  status: 'normal' | 'repairing' | 'broken'
}

export interface ItemInstance {
  id: number
  itemId: string
  name?: string
  kind?: string
  category?: string
  price?: number
  weight?: number
  slots?: number
  merchantCategory?: string
  repRequirement?: number
  currentDurability: number
  maxDurability: number
  status: 'normal' | 'depleted' | 'locked'
  locationType: string
  locationRef: string
  raidExtract: boolean
  purposes?: string[]
}

export interface RecoveryTask {
  id: number
  recoveryPlanId: number
  resourceType: 'hp' | 'energy' | 'hydration' | string
  currentValue: number
  targetValue: number
  ratePerHour: number
  status: string
  actualMethod: string
}

export interface RecoveryView {
  plan: { id: number; status: string }
  tasks: RecoveryTask[]
}

export interface Consumable {
  id: string
  name: string
  kind: 'consumable' | 'loot'
  category: string
  desc: string
  price: number
  weight: number
  slots: number
  merchantCategory: string
  repRequirement: number
  hpRecovery: number
  energyRecovery: number
  hydrationRecovery: number
  repairValue: number
  fuelSeconds: number
  maxDurability: number
  useDurability: number
  instanceRequired: boolean
  usableInSession: boolean
  usableInHideout: boolean
}

export interface GameMap {
  id: string
  name: string
  desc: string
  startNodeId: string
  layoutColumns: number
  layoutRows: number
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
  positionX: number
  positionY: number
  exploreTime: number
  distance: 'close' | 'mid' | 'far'
  enemyId: string
  encounterRole: string
  containerSlots: number
  valueTier: number
  tags: string[]
  containers: NodeContainerView[]
}

export interface MapEdge {
  id: number
  mapId: string
  fromNodeId: string
  toNodeId: string
  moveTime: number
  bidirectional: boolean
}

export interface ExtractionPoint {
  id: string
  mapId: string
  name: string
  kind: string
  anchorNodeId: string
  travelTime: number
  enabled: boolean
  iconKey: string
  tags: string[]
}

export interface MapGraph {
  map: GameMap
  nodes: MapNode[]
  edges: MapEdge[]
  extractionPoints: ExtractionPoint[]
}

export interface Enemy {
  id: string
  name: string
  kind: string
  spawnTags: string[]
  tier: number
  hpBase: number
  hpFlux: number
  hpFloor: number
  hpCap: number
  stressBase: number
  stressFlux: number
  stressFloor: number
  stressCap: number
  perceptionBase: number
  stealthBase: number
  agilityBase: number
  evasionBase: number
  mobilityBase: number
  suppressBase: number
  weaponPool: { ref: string; weight: number }[]
  armorPool: { ref: string; weight: number }[]
  ammoLevelMin: number
  ammoLevelMax: number
  ammoRoundsBase: number
  ammoRoundsMult: number
  backpackPool: { ref: string; weight: number }[]
  bossWeaponId: string
  bossArmorId: string
  bossName: string
  sortOrder: number
}


export type {
  AmmoCell,
  InventoryItem,
  Merchant,
  MerchantCatalogItem,
  MerchantCatalogResponse,
  KeyCase,
  SecureContainer,
  KeySlotView,
  PlayerLoadout,
  SaveLoadoutRequest,
  PresetSlot,
  Session,
  SessionEventType,
  SessionEvent,
  SessionRunRaw,
  SessionRun,
  SessionDetail,
  StartSessionRequest,
  RecoveryChoice,
  RecoveryMethod,
  RecoveryPolicy,
} from './types_inventory'
export type { ActionStyle, LootSummary } from './types_common'
export { presetOf } from './types_inventory'
