// 探索部署逻辑：管理地图、风格、预设装备和启动请求。
import { computed, ref, watchEffect } from 'vue'
import { ElMessage } from 'element-plus'
import api, { getApiError } from '../api'
import type {
  ActionStyle,
  Ammo,
  Armor,
  Consumable,
  GameMap,
  InventoryItem,
  Player,
  PlayerLoadout,
  Session,
  StartSessionRequest,
  Weapon,
} from '../types'
import { presetOf } from '../types'

export interface ExploreProps {
  player: Player
  loadout: PlayerLoadout
  maps: GameMap[]
  weapons: Weapon[]
  ammos: Ammo[]
  armors: Armor[]
  consumables: Consumable[]
  inventory: InventoryItem[]
}

export type ExploreEmit = (event: 'created', session: Session) => void

export function useExploreDeployment(props: ExploreProps, emit: ExploreEmit) {

  const styles = [
    { value: 'balanced' as ActionStyle, label: '均衡型', desc: '在收益与风险之间保持平衡，普通巡逻优先绕行' },
    { value: 'stealth' as ActionStyle, label: '隐秘型', desc: '优先避战、低热度和安全容器，较早返回撤离路线' },
    { value: 'aggressive' as ActionStyle, label: '激进型', desc: '主动伏击和清除敌人，接受更高战斗与热度代价' },
    { value: 'greedy' as ActionStyle, label: '贪婪型', desc: '优先高价值物资与情报，愿意为收益延后撤离' },
  ]
  
  const selectedMap = ref('')
  const selectedStyle = ref<ActionStyle>('balanced')
  const selectedPreset = ref(1)
  const selectedAmmoId = ref('')
  const selectedAmmoRounds = ref(0)
  const starting = ref(false)
  
  const currentWeapon = computed(() => props.weapons.find((item) => item.id === props.loadout.weaponId))
  const compatibleOwnedAmmos = computed(() => {
    const caliberId = currentWeapon.value?.caliberId
    if (!caliberId) return []
    return props.ammos
      .filter((ammo) => ammo.caliberId === caliberId && ammoInventoryQuantity(ammo.id) > 0)
      .sort((left, right) => right.level - left.level)
  })
  const selectedAmmoStock = computed(() => ammoInventoryQuantity(selectedAmmoId.value))
  
  function ammoInventoryQuantity(ammoId: string) {
    return props.inventory
      .filter((item) => item.kind === 'ammo' && item.itemId === ammoId)
      .reduce((total, item) => total + item.quantity, 0)
  }
  
  watchEffect(() => {
    if (!props.maps.some((item) => item.id === selectedMap.value)) selectedMap.value = props.maps[0]?.id ?? ''
  
  
    const weapon = currentWeapon.value
    if (!weapon || weapon.ammoPerRound <= 0) {
      selectedAmmoId.value = ''
      selectedAmmoRounds.value = 0
      return
    }
    const selectedIsCompatible = compatibleOwnedAmmos.value.some((item) => item.id === selectedAmmoId.value)
    if (!selectedIsCompatible) selectedAmmoId.value = compatibleOwnedAmmos.value[0]?.id ?? ''
    const stock = ammoInventoryQuantity(selectedAmmoId.value)
    if (stock <= 0) {
      selectedAmmoRounds.value = 0
      return
    }
    if (selectedAmmoRounds.value < weapon.ammoPerRound || selectedAmmoRounds.value > stock) {
      selectedAmmoRounds.value = Math.min(120, stock)
    }
  })
  
  function presetSummary(index: number) {
    const slot = presetOf(props.loadout, index)
    const weapon = props.weapons.find((item) => item.id === slot.weaponId)
    const armor = props.armors.find((item) => item.id === slot.armorId)
    if (!slot.weaponId || !slot.armorId || !weapon || !armor) return '未配置，请先在角色页面设置'
    const supplies = slot.consumables.map((id) => props.consumables.find((item) => item.id === id)?.name ?? id).join('、')
    const ammo = props.ammos.find((item) => item.id === slot.ammoId)
    const ammoText = weapon.ammoPerRound > 0 ? ` · ${ammo?.name ?? '未配置弹药'} ×${slot.ammoRounds}` : ''
    return `${weapon.name} · ${armor.name}${ammoText}${supplies ? ` · ${supplies}` : ' · 无补给'}`
  }
  
  const selectedPresetSummary = computed(() => presetSummary(selectedPreset.value))
  const canSubmit = computed(() => {
    const preset = presetOf(props.loadout, selectedPreset.value)
    const weapon = currentWeapon.value
    const hasCompatibleAmmo = !weapon || weapon.ammoPerRound <= 0 || Boolean(
      selectedAmmoId.value
      && selectedAmmoRounds.value >= weapon.ammoPerRound
      && selectedAmmoRounds.value <= selectedAmmoStock.value,
    )
    return Boolean(
      selectedMap.value && props.loadout.weaponId && props.loadout.armorId
      && preset.weaponId && preset.armorId && hasCompatibleAmmo,
    )
  })
  
  function buildRequest(): StartSessionRequest {
    return {
      mapId: selectedMap.value,
      style: selectedStyle.value,
      recoveryPreset: selectedPreset.value,
      ammoId: selectedAmmoId.value,
      ammoRounds: selectedAmmoRounds.value,
    }
  }
  
  async function startSession() {
    starting.value = true
    try {
      const { data } = await api.post<Session>('/session/start', buildRequest())
      ElMessage.success(`行动 #${data.id} 已开始`)
      emit('created', data)
    } catch (error) {
      ElMessage.error(getApiError(error, '行动启动失败'))
    } finally {
      starting.value = false
    }
  }

  return {
    styles, selectedMap, selectedStyle, selectedPreset, selectedAmmoId, selectedAmmoRounds, starting,
    currentWeapon, compatibleOwnedAmmos, ammoInventoryQuantity, selectedAmmoStock,
    presetSummary, selectedPresetSummary, canSubmit, startSession,
  }
}
