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

export interface User {
	id: number
	username: string
	email?: string | null
	status: string
	createdAt: string
	updatedAt: string
}

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
  id: string
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
  hp: number
  stressThreshold: number
  perception: number
  stealth: number
  agility: number
  weaponId: string
  armorId: string
  ammoId: string
  ammoRounds: number
  evasion: number
  mobility: number
  suppress: number
  backpackContainerId: string
}


export type {
  InventoryItem,
  Merchant,
  MerchantCatalogItem,
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
} from './types_inventory'
export { presetOf } from './types_inventory'
