// 角色装备基础模型：集中管理部位定义、装备集合、随身弹药槽和预设 loadout 映射。
import type { PlayerLoadout } from '../types'
import type { AmmoCell } from '../types'

export type SlotKey = 'weaponId' | 'armorId' | 'chestRigId' | 'backpackId' | 'helmetId' | 'headsetId' | 'keyCaseId' | 'secureContainerId'

export interface SlotDef { key: SlotKey; label: string; row: 1 | 2 | 3; col: 1 | 2 | 3 }
export type EquipSet = { [k in SlotKey]: string } & {
  consumables: string[]
  keyInstanceIds: number[]
  ammo: AmmoCell[]
  name: string
  ammoId: string
  ammoRounds: number
}

export const slotDefs: SlotDef[] = [
  { key: 'headsetId', label: '耳机', row: 1, col: 1 },
  { key: 'helmetId', label: '头盔', row: 1, col: 2 },
  { key: 'chestRigId', label: '胸挂', row: 1, col: 3 },
  { key: 'weaponId', label: '武器', row: 2, col: 1 },
  { key: 'armorId', label: '护甲', row: 2, col: 2 },
  { key: 'backpackId', label: '背包', row: 2, col: 3 },
  { key: 'keyCaseId', label: '钥匙包', row: 3, col: 1 },
  { key: 'secureContainerId', label: '安全箱', row: 3, col: 2 },
]
export const gridRows = [1, 2, 3] as const

// 随身弹药最大槽位数与单格最大发数（与后端校验一致）。
export const ammoSlotCount = 4
export const ammoSlotMaxRounds = 60

export function emptyAmmoCells(): AmmoCell[] {
  return Array.from({ length: ammoSlotCount }, () => ({ ammoId: '', rounds: 0 }))
}

export function ammoCellsFromLoadout(cells?: AmmoCell[] | null): AmmoCell[] {
  const out = emptyAmmoCells()
  ;(cells ?? []).slice(0, ammoSlotCount).forEach((cell, index) => {
    out[index] = { ammoId: cell.ammoId ?? '', rounds: cell.rounds ?? 0 }
  })
  return out
}

export function emptySet(name = ''): EquipSet {
  return {
    weaponId: '', armorId: '', chestRigId: '', backpackId: '', helmetId: '', headsetId: '', keyCaseId: '', secureContainerId: '',
    consumables: [], keyInstanceIds: [], ammo: emptyAmmoCells(), name, ammoId: '', ammoRounds: 0,
  }
}

export function fromLoadout(l: PlayerLoadout): EquipSet {
  return {
    weaponId: l.weaponId, armorId: l.armorId, chestRigId: l.chestRigId, backpackId: l.backpackId,
    helmetId: l.helmetId, headsetId: l.headsetId, keyCaseId: l.keyCaseId ?? '', secureContainerId: l.secureContainerId ?? '',
    consumables: [...l.consumables],
    keyInstanceIds: (l.keys ?? []).map((slot) => slot.instanceId || 0),
    ammo: ammoCellsFromLoadout(l.carriedAmmo),
    name: '', ammoId: '', ammoRounds: 0,
  }
}
export function presetFromLoadout(l: PlayerLoadout, index: number): EquipSet {
  if (index === 0) {
    return {
      weaponId: l.presetWeaponId, armorId: l.presetArmorId, chestRigId: l.presetChestRigId, backpackId: l.presetBackpackId,
      helmetId: l.presetHelmetId, headsetId: l.presetHeadsetId, keyCaseId: '', secureContainerId: '',
      consumables: [...l.presetConsumables], keyInstanceIds: [], ammo: emptyAmmoCells(), name: l.presetName,
      ammoId: l.presetAmmoId, ammoRounds: l.presetAmmoRounds,
    }
  }
  if (index === 1) {
    return {
      weaponId: l.preset2WeaponId, armorId: l.preset2ArmorId, chestRigId: l.preset2ChestRigId, backpackId: l.preset2BackpackId,
      helmetId: l.preset2HelmetId, headsetId: l.preset2HeadsetId, keyCaseId: '', secureContainerId: '',
      consumables: [...l.preset2Consumables], keyInstanceIds: [], ammo: emptyAmmoCells(), name: l.preset2Name,
      ammoId: l.preset2AmmoId, ammoRounds: l.preset2AmmoRounds,
    }
  }
  return {
    weaponId: l.preset3WeaponId, armorId: l.preset3ArmorId, chestRigId: l.preset3ChestRigId, backpackId: l.preset3BackpackId,
    helmetId: l.preset3HelmetId, headsetId: l.preset3HeadsetId, keyCaseId: '', secureContainerId: '',
    consumables: [...l.preset3Consumables], keyInstanceIds: [], ammo: emptyAmmoCells(), name: l.preset3Name,
    ammoId: l.preset3AmmoId, ammoRounds: l.preset3AmmoRounds,
  }
}
