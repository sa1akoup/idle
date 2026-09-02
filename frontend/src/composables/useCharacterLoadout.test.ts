// 角色装备保存行为回归测试：自动保存已移除，装备调整只走显式保存按钮；
// 未保存状态（hasUnsavedChanges）供离开角色页前的提示使用，并防止轮询刷新吞掉未保存的编辑。
import { nextTick, reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import type { PlayerLoadout, SaveLoadoutRequest } from '../types'
import { useCharacterLoadout } from './useCharacterLoadout'

function makeLoadout(overrides: Partial<PlayerLoadout> = {}): PlayerLoadout {
  return {
    id: 1,
    characterId: 1,
    weaponId: 'rifle_ak',
    armorId: 'light_01',
    chestRigId: '',
    backpackId: '',
    helmetId: '',
    headsetId: '',
    consumables: [],
    presetWeaponId: '',
    presetArmorId: '',
    presetChestRigId: '',
    presetBackpackId: '',
    presetHelmetId: '',
    presetHeadsetId: '',
    presetName: '',
    presetConsumables: [],
    presetAmmoId: '',
    presetAmmoRounds: 0,
    preset2WeaponId: '',
    preset2ArmorId: '',
    preset2ChestRigId: '',
    preset2BackpackId: '',
    preset2HelmetId: '',
    preset2HeadsetId: '',
    preset2Name: '',
    preset2Consumables: [],
    preset2AmmoId: '',
    preset2AmmoRounds: 0,
    preset3WeaponId: '',
    preset3ArmorId: '',
    preset3ChestRigId: '',
    preset3BackpackId: '',
    preset3HelmetId: '',
    preset3HeadsetId: '',
    preset3Name: '',
    preset3Consumables: [],
    preset3AmmoId: '',
    preset3AmmoRounds: 0,
    updatedAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

function invoke(loadoutOverrides: Partial<PlayerLoadout> = {}) {
	const emitted: Array<[string, SaveLoadoutRequest, boolean?]> = []
	const props = reactive({
    player: {
      name: '测试角色', strength: 50, hp: 100, hpMax: 100, stress: 0, stressMax: 100,
      energy: 100, hydration: 100, recoveryPerHour: { hp: 0, stress: 0, energy: 0, hydration: 0 },
    },
    loadout: makeLoadout(loadoutOverrides),
    inventory: [],
    weapons: [],
    ammos: [],
    merchants: [],
    armors: [],
    consumables: [],
    itemInstances: [],
    chestRigs: [],
    backpacks: [],
    helmets: [],
    headsets: [],
    savingName: false,
    savingLoadout: false,
  })
  const api = useCharacterLoadout(
    props as never,
    ((event: string, request: SaveLoadoutRequest, silent?: boolean) => {
      emitted.push([event, request, silent])
    }) as never,
  )
  return {
    props, emitted, current: api.current, presets: api.presets, hasUnsavedChanges: api.hasUnsavedChanges,
    submitLoadout: api.submitLoadout, ammoCellMaxRounds: api.ammoCellMaxRounds,
  }
}

describe('useCharacterLoadout 装备保存行为', () => {
  it('调整装备后不会自动保存，只有显式提交才发出 saveLoadout', async () => {
    const { emitted, current, hasUnsavedChanges, submitLoadout } = invoke()
    await nextTick()
    expect(emitted).toHaveLength(0)
    expect(hasUnsavedChanges.value).toBe(false)

    current.value.weaponId = 'rifle_m4a1'
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)
    // 无自动保存：等待超过原防抖时长也不应产生请求
    await new Promise((resolve) => setTimeout(resolve, 50))
    await nextTick()
    expect(emitted).toHaveLength(0)

    // 显式点击"保存装备配置"才触发保存
    submitLoadout()
    expect(emitted).toHaveLength(1)
    expect(emitted[0][1].weaponId).toBe('rifle_m4a1')
  })

  it('保存成功后的 loadout 回刷会清掉未保存状态；保存失败时保留编辑并保持未保存', async () => {
    const { props, current, hasUnsavedChanges } = invoke()
    await nextTick()

    current.value.weaponId = 'rifle_m4a1'
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)

    // 模拟保存成功：后端回刷内容与当前编辑一致
    props.loadout = makeLoadout({ weaponId: 'rifle_m4a1' })
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(false)

    // 模拟保存失败：后端回刷旧内容，但用户编辑应保留、未保存状态保持（供离开提示使用）
    current.value.armorId = 'heavy_01'
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)
    props.loadout = makeLoadout()
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)
    expect(current.value.armorId).toBe('heavy_01')

    // 用户手动改回与后端一致后，未保存状态才清零
    current.value.armorId = 'light_01'
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(false)
  })

  it('有未保存修改时，轮询刷新不会覆盖正在编辑的内容', async () => {
    const { props, current, hasUnsavedChanges } = invoke()
    await nextTick()

    current.value.weaponId = 'rifle_m4a1'
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)

    // 后台轮询回刷后端旧数据：current 应保留用户编辑，未保存状态不丢失
    props.loadout = makeLoadout()
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)
    expect(current.value.weaponId).toBe('rifle_m4a1')

    // 无未保存修改时，刷新正常生效
    current.value.weaponId = 'rifle_ak'
    props.loadout = makeLoadout({ weaponId: 'rifle_m4a1' })
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(false)
    expect(current.value.weaponId).toBe('rifle_m4a1')
  })

	it('预设修改也会触发未保存提示，轮询刷新不会覆盖预设编辑', async () => {
    const { props, presets, hasUnsavedChanges } = invoke()
    await nextTick()

    presets.value[0].name = '夜间突击'
    presets.value[0].ammoRounds = 45
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)

    props.loadout = makeLoadout()
    await nextTick()
    expect(hasUnsavedChanges.value).toBe(true)
    expect(presets.value[0].name).toBe('夜间突击')
    expect(presets.value[0].ammoRounds).toBe(45)

    props.loadout = makeLoadout({ presetName: '夜间突击', presetAmmoRounds: 45 })
    await nextTick()
		expect(hasUnsavedChanges.value).toBe(false)
	})

	it('已预留弹药的当前配置不会因仓库库存归零而降低上限', async () => {
		const { current, ammoCellMaxRounds } = invoke({
			carriedAmmo: [{ ammoId: 'ammo-reserved', rounds: 60 }],
		})
		await nextTick()

		expect(ammoCellMaxRounds(current.value.ammo[0].ammoId, current.value.ammo[0].rounds)).toBe(60)
		expect(ammoCellMaxRounds('', 0)).toBe(0)
	})
})
