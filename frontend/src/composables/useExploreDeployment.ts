// 探索部署逻辑：管理地图、风格、失能预案与启动请求。
// 携带弹药不再在此配置：弹药槽统一在角色页设置，开局按槽位从仓库预留。
import { computed, ref, watchEffect } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import api, { getApiError } from '../api'
import type {
  ActionStyle,
  Ammo,
  Armor,
  GameMap,
  InventoryItem,
  Player,
  PlayerLoadout,
  RecoveryMethod,
  RecoveryPolicy,
  RecoveryView,
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
  inventory: InventoryItem[]
  recovery: RecoveryView | null
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
  const selectedHPRecoveryMethod = ref<RecoveryMethod>('inventory')
  const selectedEnergyRecoveryMethod = ref<RecoveryMethod>('inventory')
  const selectedHydrationRecoveryMethod = ref<RecoveryMethod>('inventory')
  const starting = ref(false)

  const recoveryMethods: Array<{ value: RecoveryMethod; label: string }> = [
    { value: 'inventory', label: '优先使用库存' },
    { value: 'hideout', label: '藏身处等待' },
    { value: 'merchant', label: '商人购买' },
  ]

  const recoveryPending = computed(() => Boolean(
    props.recovery?.plan.status === 'running'
    && props.recovery.tasks.some((task) => task.status !== 'completed' && task.currentValue < task.targetValue - 0.000001),
  ))

  const selectedPresetSlot = computed(() => presetOf(props.loadout, selectedPreset.value))
  // 当前携行只反映真实装备：无预设兜底，未装备即显示空。
  const currentWeapon = computed(() => props.weapons.find((item) => item.id === props.loadout.weaponId))
  const currentArmor = computed(() => props.armors.find((item) => item.id === props.loadout.armorId))
  const hasCurrentWeapon = computed(() => Boolean(currentWeapon.value))
  const hasCurrentArmor = computed(() => Boolean(currentArmor.value))
  const currentLoadoutIsEmpty = computed(() => Boolean(
    !props.loadout.weaponId && !props.loadout.armorId && !props.loadout.chestRigId && !props.loadout.backpackId
    && !props.loadout.helmetId && !props.loadout.headsetId && props.loadout.consumables.length === 0,
  ))

  watchEffect(() => {
    if (!props.maps.some((item) => item.id === selectedMap.value)) selectedMap.value = props.maps[0]?.id ?? ''
  })

  // 失能预案卡片只显示预设名称（数据为保存快照，不实时展示明细）。
  function presetLabel(index: number): string {
    const slot = presetOf(props.loadout, index)
    return slot.name?.trim() ? slot.name.trim() : `预设 ${index}`
  }

  function recoveryMethodLabel(method: RecoveryMethod): string {
    return recoveryMethods.find((item) => item.value === method)?.label ?? method
  }

  function fallbackMethodFor(method: RecoveryMethod): RecoveryMethod | 'none' {
    if (method === 'inventory') return 'hideout'
    if (method === 'hideout') return 'none'
    return 'hideout'
  }

  function buildRecoveryPolicy(): RecoveryPolicy {
    return {
      hp: { targetPercent: 100, primaryMethod: selectedHPRecoveryMethod.value, fallbackMethod: fallbackMethodFor(selectedHPRecoveryMethod.value) },
      energy: { targetPercent: 80, primaryMethod: selectedEnergyRecoveryMethod.value, fallbackMethod: fallbackMethodFor(selectedEnergyRecoveryMethod.value) },
      hydration: { targetPercent: 80, primaryMethod: selectedHydrationRecoveryMethod.value, fallbackMethod: fallbackMethodFor(selectedHydrationRecoveryMethod.value) },
      merchantEnable: true,
    }
  }

  const canSubmit = computed(() => {
    const preset = selectedPresetSlot.value
    return Boolean(
      selectedMap.value && preset.weaponId && preset.armorId
      && (Boolean(props.loadout.weaponId) || currentLoadoutIsEmpty.value),
    )
  })

  function buildRequest(): StartSessionRequest {
    return {
      mapId: selectedMap.value,
      style: selectedStyle.value,
      recoveryPreset: selectedPreset.value,
      recoveryPolicy: buildRecoveryPolicy(),
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
    styles, selectedMap, selectedStyle, selectedPreset, starting,
    recoveryMethods, selectedHPRecoveryMethod, selectedEnergyRecoveryMethod, selectedHydrationRecoveryMethod,
    currentWeapon, currentArmor, hasCurrentWeapon, hasCurrentArmor, recoveryPending,
    presetLabel, recoveryMethodLabel, canSubmit, startSession,
  }
}
