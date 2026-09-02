// 角色装备辅助函数测试：覆盖空装备、当前装备（含随身弹药）、三套预设的字段映射及数组隔离。
import { describe, expect, it } from 'vitest'
import type { PlayerLoadout } from '../types'
import { emptySet, emptyAmmoCells, fromLoadout, presetFromLoadout } from './characterLoadoutHelpers'

function makeLoadout(): PlayerLoadout {
  return {
    id: 1,
    characterId: 2,
    weaponId: 'weapon-current',
    armorId: 'armor-current',
    chestRigId: 'chest-current',
    backpackId: 'backpack-current',
    helmetId: 'helmet-current',
    headsetId: 'headset-current',
    consumables: ['consumable-current'],
    carriedAmmo: [
      { ammoId: 'ammo_cal_n2', rounds: 30 },
      { ammoId: 'ammo_cal_n2', rounds: 30 },
    ],
    presetWeaponId: 'weapon-preset-1',
    presetArmorId: 'armor-preset-1',
    presetChestRigId: 'chest-preset-1',
    presetBackpackId: 'backpack-preset-1',
    presetHelmetId: 'helmet-preset-1',
    presetHeadsetId: 'headset-preset-1',
    presetConsumables: ['consumable-preset-1'],
    presetAmmoId: 'ammo-preset-1',
    presetAmmoRounds: 30,
    presetName: '预设一',
    preset2WeaponId: 'weapon-preset-2',
    preset2ArmorId: 'armor-preset-2',
    preset2ChestRigId: 'chest-preset-2',
    preset2BackpackId: 'backpack-preset-2',
    preset2HelmetId: 'helmet-preset-2',
    preset2HeadsetId: 'headset-preset-2',
    preset2Consumables: ['consumable-preset-2'],
    preset2AmmoId: 'ammo-preset-2',
    preset2AmmoRounds: 60,
    preset2Name: '预设二',
    preset3WeaponId: 'weapon-preset-3',
    preset3ArmorId: 'armor-preset-3',
    preset3ChestRigId: 'chest-preset-3',
    preset3BackpackId: 'backpack-preset-3',
    preset3HelmetId: 'helmet-preset-3',
    preset3HeadsetId: 'headset-preset-3',
    preset3Consumables: ['consumable-preset-3'],
    preset3AmmoId: 'ammo-preset-3',
    preset3AmmoRounds: 90,
    preset3Name: '预设三',
    updatedAt: '2026-08-31T00:00:00Z',
  }
}

describe('characterLoadoutHelpers', () => {
  it('creates an empty equipment set with an optional name', () => {
    expect(emptySet('空白预设')).toEqual({
      weaponId: '',
      armorId: '',
      chestRigId: '',
      backpackId: '',
      helmetId: '',
      headsetId: '',
      consumables: [],
      ammo: emptyAmmoCells(),
      name: '空白预设',
      ammoId: '',
      ammoRounds: 0,
    })
  })

  it('maps the current loadout fields', () => {
    const loadout = makeLoadout()

    expect(fromLoadout(loadout)).toEqual({
      weaponId: 'weapon-current',
      armorId: 'armor-current',
      chestRigId: 'chest-current',
      backpackId: 'backpack-current',
      helmetId: 'helmet-current',
      headsetId: 'headset-current',
      consumables: ['consumable-current'],
      ammo: [
        { ammoId: 'ammo_cal_n2', rounds: 30 },
        { ammoId: 'ammo_cal_n2', rounds: 30 },
        { ammoId: '', rounds: 0 },
        { ammoId: '', rounds: 0 },
      ],
      name: '',
      ammoId: '',
      ammoRounds: 0,
    })
  })

  it('maps all three preset layouts', () => {
    const loadout = makeLoadout()

    expect(presetFromLoadout(loadout, 0)).toEqual({
      weaponId: 'weapon-preset-1',
      armorId: 'armor-preset-1',
      chestRigId: 'chest-preset-1',
      backpackId: 'backpack-preset-1',
      helmetId: 'helmet-preset-1',
      headsetId: 'headset-preset-1',
      consumables: ['consumable-preset-1'],
      ammo: emptyAmmoCells(),
      name: '预设一',
      ammoId: 'ammo-preset-1',
      ammoRounds: 30,
    })
    expect(presetFromLoadout(loadout, 1)).toEqual({
      weaponId: 'weapon-preset-2',
      armorId: 'armor-preset-2',
      chestRigId: 'chest-preset-2',
      backpackId: 'backpack-preset-2',
      helmetId: 'helmet-preset-2',
      headsetId: 'headset-preset-2',
      consumables: ['consumable-preset-2'],
      ammo: emptyAmmoCells(),
      name: '预设二',
      ammoId: 'ammo-preset-2',
      ammoRounds: 60,
    })
    expect(presetFromLoadout(loadout, 2)).toEqual({
      weaponId: 'weapon-preset-3',
      armorId: 'armor-preset-3',
      chestRigId: 'chest-preset-3',
      backpackId: 'backpack-preset-3',
      helmetId: 'helmet-preset-3',
      headsetId: 'headset-preset-3',
      consumables: ['consumable-preset-3'],
      ammo: emptyAmmoCells(),
      name: '预设三',
      ammoId: 'ammo-preset-3',
      ammoRounds: 90,
    })
  })

  it('copies consumable arrays so edits do not mutate the loadout', () => {
    const loadout = makeLoadout()
    const current = fromLoadout(loadout)
    const presets = [0, 1, 2].map((index) => presetFromLoadout(loadout, index))

    current.consumables.push('current-only')
    presets[0].consumables.push('preset-1-only')
    presets[1].consumables[0] = 'preset-2-only'
    presets[2].consumables.splice(0, 1)

    expect(loadout.consumables).toEqual(['consumable-current'])
    expect(loadout.presetConsumables).toEqual(['consumable-preset-1'])
    expect(loadout.preset2Consumables).toEqual(['consumable-preset-2'])
    expect(loadout.preset3Consumables).toEqual(['consumable-preset-3'])
    expect(current.consumables).not.toBe(loadout.consumables)
    expect(presets[0].consumables).not.toBe(loadout.presetConsumables)
    expect(presets[1].consumables).not.toBe(loadout.preset2Consumables)
    expect(presets[2].consumables).not.toBe(loadout.preset3Consumables)
  })
})
