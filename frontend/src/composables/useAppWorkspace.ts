// 应用工作区状态：集中加载全局数据、处理用户操作并维护功能视图。
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import api, { getApiError, isUnauthorized, setUnauthorizedHandler } from '../api'
import type {
  Ammo,
  Armor,
  ArmorInstance,
  Backpack,
  ChestRig,
  Consumable,
  CraftingRecipe,
  Enemy,
  GameMap,
  HideoutSnapshot,
  Headset,
  Helmet,
  InventoryItem,
  ItemInstance,
  MapGraph,
  Merchant,
  NavKey,
  Player,
  PlayerLoadout,
  RecoveryView,
  SaveLoadoutRequest,
  Session,
  StorageCapacity,
  User,
  Weapon,
} from '../types'

type ResourceKey =
  | 'player'
  | 'sessions'
  | 'maps'
  | 'enemies'
  | 'weapons'
  | 'ammos'
  | 'armors'
  | 'armorInstances'
  | 'itemInstances'
  | 'consumables'
  | 'chestRigs'
  | 'backpacks'
  | 'helmets'
  | 'headsets'
  | 'merchants'
  | 'inventory'
  | 'storageCapacity'
  | 'hideout'
  | 'craftingRecipes'
  | 'recovery'
  | 'loadout'

interface ViewState {
  loading: boolean
  error: string
  ready: boolean
}

const resourceKeys: ResourceKey[] = [
  'player', 'sessions', 'maps', 'enemies', 'weapons', 'ammos', 'armors', 'armorInstances', 'itemInstances',
  'consumables', 'chestRigs', 'backpacks', 'helmets', 'headsets', 'merchants', 'inventory', 'storageCapacity',
  'hideout', 'craftingRecipes', 'recovery', 'loadout',
]

const viewKeys: NavKey[] = ['explore', 'live', 'map', 'character', 'inventory', 'merchant', 'hideout', 'logs']

export function useAppWorkspace() {

  const activeView = ref<NavKey>('explore')
  const user = ref<User | null>(null)
  const authChecking = ref(true)
  const authError = ref('')
  const mobileOpen = ref(false)
  const loading = ref(true)
  const loadError = ref('')
  const savingPlayer = ref(false)
  const savingLoadout = ref(false)
  const purchasingId = ref<string | null>(null)
  const sellingId = ref<string | null>(null)
  const repairingId = ref<number | null>(null)
  const upgradingFacilityId = ref<string | null>(null)
  const craftingId = ref<string | null>(null)
  
  const player = ref<Player | null>(null)
  const loadout = ref<PlayerLoadout | null>(null)
  const maps = ref<GameMap[]>([])
  const mapGraphs = ref<Record<string, MapGraph>>({})
  const enemies = ref<Enemy[]>([])
  const weapons = ref<Weapon[]>([])
  const ammos = ref<Ammo[]>([])
  const armors = ref<Armor[]>([])
  const armorInstances = ref<ArmorInstance[]>([])
  const itemInstances = ref<ItemInstance[]>([])
  const consumables = ref<Consumable[]>([])
  const chestRigs = ref<ChestRig[]>([])
  const backpacks = ref<Backpack[]>([])
  const helmets = ref<Helmet[]>([])
  const headsets = ref<Headset[]>([])
  const merchants = ref<Merchant[]>([])
  const inventory = ref<InventoryItem[]>([])
  const storageCapacity = ref<StorageCapacity | null>(null)
  const hideout = ref<HideoutSnapshot | null>(null)
  const craftingRecipes = ref<CraftingRecipe[]>([])
  const recovery = ref<RecoveryView | null>(null)
  const sessions = ref<Session[]>([])
  const activeSessionId = ref<number | null>(null)
  let recoveryPollTimer: ReturnType<typeof setTimeout> | undefined
  let recoveryRefreshInFlight = false
  let workspaceStopped = false
  let loadoutSaveQueue: Promise<void> = Promise.resolve()
  let pendingLoadoutSaves = 0
  let loadoutSaveVersion = 0
  
  const viewTitles: Record<NavKey, string> = {
    explore: '探索部署', live: '实时行动', map: '区域地图', character: '玩家角色', inventory: '本地仓库',
    merchant: '灰区商人', hideout: '藏身处', logs: '行动日志',
  }

  const viewState = reactive<Record<NavKey, ViewState>>({
    explore: { loading: false, error: '', ready: false },
    live: { loading: false, error: '', ready: false },
    map: { loading: false, error: '', ready: false },
    character: { loading: false, error: '', ready: false },
    inventory: { loading: false, error: '', ready: false },
    merchant: { loading: false, error: '', ready: false },
    hideout: { loading: false, error: '', ready: false },
    logs: { loading: false, error: '', ready: false },
  })
  const resourceLoaded = reactive<Record<ResourceKey, boolean>>(
    Object.fromEntries(resourceKeys.map((key) => [key, false])) as Record<ResourceKey, boolean>,
  )
  const resourcePromises = new Map<ResourceKey, Promise<void>>()
  const graphPromises = new Map<string, Promise<void>>()
  const viewPromises = new Map<NavKey, Promise<void>>()
  let workspaceGeneration = 0
  
  const cash = computed(() => inventory.value.find((item) => item.itemId === 'cash')?.quantity ?? 0)
  const latestSession = computed(() => sessions.value[0] ?? null)
  const activeSession = computed(() => sessions.value.find((item) => item.status === 'running') ?? null)
  
  setUnauthorizedHandler(() => {
    resetWorkspaceData()
    user.value = null
  })

  function resetWorkspaceData() {
    workspaceGeneration += 1
    resourcePromises.clear()
    graphPromises.clear()
    viewPromises.clear()
    recoveryRefreshInFlight = false
    for (const key of resourceKeys) resourceLoaded[key] = false
    for (const key of viewKeys) {
      viewState[key].loading = false
      viewState[key].error = ''
      viewState[key].ready = false
    }
    player.value = null
    loadout.value = null
    maps.value = []
    mapGraphs.value = {}
    enemies.value = []
    weapons.value = []
    ammos.value = []
    armors.value = []
    armorInstances.value = []
    itemInstances.value = []
    consumables.value = []
    chestRigs.value = []
    backpacks.value = []
    helmets.value = []
    headsets.value = []
    merchants.value = []
    inventory.value = []
    storageCapacity.value = null
    hideout.value = null
    craftingRecipes.value = []
    recovery.value = null
    sessions.value = []
    activeSessionId.value = null
    activeView.value = 'explore'
    clearRecoveryPoll()
  }

  function clearRecoveryPoll() {
    if (recoveryPollTimer !== undefined) {
      clearTimeout(recoveryPollTimer)
      recoveryPollTimer = undefined
    }
  }

  function scheduleRecoveryPoll() {
    clearRecoveryPoll()
    if (workspaceStopped || recovery.value?.plan.status !== 'running') return
    recoveryPollTimer = setTimeout(() => {
      recoveryPollTimer = undefined
      void refreshRecoveryState(true)
    }, 5000)
  }

  async function refreshRecoveryState(silent = true) {
    if (workspaceStopped || recoveryRefreshInFlight) return
    const generation = workspaceGeneration
    recoveryRefreshInFlight = true
    try {
      const [recoveryRes, playerRes] = await Promise.all([
        api.get<RecoveryView | null>('/recovery/current'), api.get<Player>('/player'),
      ])
      if (workspaceStopped || generation !== workspaceGeneration) return
      recovery.value = recoveryRes.data
      player.value = playerRes.data
      resourceLoaded.recovery = true
      resourceLoaded.player = true
    } catch (error) {
      if (generation === workspaceGeneration && !silent) ElMessage.error(getApiError(error, '恢复状态刷新失败'))
    } finally {
      if (generation === workspaceGeneration) {
        recoveryRefreshInFlight = false
        scheduleRecoveryPoll()
      }
    }
  }

  async function ensureResource<T>(
    key: ResourceKey,
    request: () => Promise<{ data: T }>,
    assign: (data: T) => void,
  ): Promise<void> {
    if (resourceLoaded[key]) return
    const pending = resourcePromises.get(key)
    if (pending) return pending

    const generation = workspaceGeneration
    const task = (async () => {
      const response = await request()
      if (workspaceStopped || generation !== workspaceGeneration) return
      assign(response.data)
      resourceLoaded[key] = true
    })()
    resourcePromises.set(key, task)
    const cleanup = () => {
      if (resourcePromises.get(key) === task) resourcePromises.delete(key)
    }
    void task.then(cleanup, cleanup)
    return task
  }

  async function ensureMapGraph(mapId: string): Promise<void> {
    if (mapGraphs.value[mapId]) return
    const pending = graphPromises.get(mapId)
    if (pending) return pending

    const generation = workspaceGeneration
    const task = (async () => {
      const { data } = await api.get<MapGraph>(`/maps/${mapId}/graph`)
      if (workspaceStopped || generation !== workspaceGeneration) return
      mapGraphs.value = { ...mapGraphs.value, [data.map.id]: data }
    })()
    graphPromises.set(mapId, task)
    const cleanup = () => {
      if (graphPromises.get(mapId) === task) graphPromises.delete(mapId)
    }
    void task.then(cleanup, cleanup)
    return task
  }

  async function ensureMapGraphs(mapIds: string[]) {
    await Promise.all(mapIds.map((mapId) => ensureMapGraph(mapId)))
  }

  function markResourcesLoaded(...keys: ResourceKey[]) {
    for (const key of keys) resourceLoaded[key] = true
  }

  async function loadCoreData() {
    await Promise.all([
      ensureResource('player', () => api.get<Player>('/player'), (data) => { player.value = data }),
      ensureResource('sessions', () => api.get<Session[]>('/sessions'), (data) => { sessions.value = data }),
    ])
    activeSessionId.value = sessions.value.find((item) => item.status === 'running')?.id ?? null
  }

  async function loadExploreData() {
    await Promise.all([
      ensureResource('maps', () => api.get<GameMap[]>('/maps'), (data) => { maps.value = data }),
      ensureResource('loadout', () => api.get<PlayerLoadout>('/loadout'), (data) => { loadout.value = data }),
      ensureResource('weapons', () => api.get<Weapon[]>('/weapons'), (data) => { weapons.value = data }),
      ensureResource('ammos', () => api.get<Ammo[]>('/ammos'), (data) => { ammos.value = data }),
      ensureResource('armors', () => api.get<Armor[]>('/armors'), (data) => { armors.value = data }),
      ensureResource('consumables', () => api.get<Consumable[]>('/consumables'), (data) => { consumables.value = data }),
      ensureResource('inventory', () => api.get<InventoryItem[]>('/inventory'), (data) => { inventory.value = data }),
      ensureResource('recovery', () => api.get<RecoveryView | null>('/recovery/current'), (data) => { recovery.value = data }),
    ])
    scheduleRecoveryPoll()
  }

  async function loadMapData() {
    await Promise.all([
      ensureResource('maps', () => api.get<GameMap[]>('/maps'), (data) => { maps.value = data }),
      ensureResource('enemies', () => api.get<Enemy[]>('/enemies'), (data) => { enemies.value = data }),
    ])
    await ensureMapGraphs(maps.value.map((map) => map.id))
  }

  async function loadLiveData() {
    await ensureResource('maps', () => api.get<GameMap[]>('/maps'), (data) => { maps.value = data })
    const session = sessions.value.find((item) => item.id === activeSessionId.value)
    if (session) await ensureMapGraphs([session.mapId])
  }

  async function loadCharacterData() {
    await Promise.all([
      ensureResource('loadout', () => api.get<PlayerLoadout>('/loadout'), (data) => { loadout.value = data }),
      ensureResource('inventory', () => api.get<InventoryItem[]>('/inventory'), (data) => { inventory.value = data }),
      ensureResource('weapons', () => api.get<Weapon[]>('/weapons'), (data) => { weapons.value = data }),
      ensureResource('ammos', () => api.get<Ammo[]>('/ammos'), (data) => { ammos.value = data }),
      ensureResource('merchants', () => api.get<Merchant[]>('/merchants'), (data) => { merchants.value = data }),
      ensureResource('armors', () => api.get<Armor[]>('/armors'), (data) => { armors.value = data }),
      ensureResource('consumables', () => api.get<Consumable[]>('/consumables'), (data) => { consumables.value = data }),
      ensureResource('itemInstances', () => api.get<ItemInstance[]>('/item-instances'), (data) => { itemInstances.value = data }),
      ensureResource('chestRigs', () => api.get<ChestRig[]>('/chestrigs'), (data) => { chestRigs.value = data }),
      ensureResource('backpacks', () => api.get<Backpack[]>('/backpacks'), (data) => { backpacks.value = data }),
      ensureResource('helmets', () => api.get<Helmet[]>('/helmets'), (data) => { helmets.value = data }),
      ensureResource('headsets', () => api.get<Headset[]>('/headsets'), (data) => { headsets.value = data }),
    ])
  }

  async function loadInventoryData() {
    await Promise.all([
      ensureResource('inventory', () => api.get<InventoryItem[]>('/inventory'), (data) => { inventory.value = data }),
      ensureResource('itemInstances', () => api.get<ItemInstance[]>('/item-instances'), (data) => { itemInstances.value = data }),
      ensureResource('storageCapacity', () => api.get<StorageCapacity>('/inventory/capacity'), (data) => { storageCapacity.value = data }),
      ensureResource('loadout', () => api.get<PlayerLoadout>('/loadout'), (data) => { loadout.value = data }),
    ])
  }

  async function loadMerchantData() {
    await Promise.all([
      ensureResource('merchants', () => api.get<Merchant[]>('/merchants'), (data) => { merchants.value = data }),
      ensureResource('inventory', () => api.get<InventoryItem[]>('/inventory'), (data) => { inventory.value = data }),
      ensureResource('itemInstances', () => api.get<ItemInstance[]>('/item-instances'), (data) => { itemInstances.value = data }),
    ])
  }

  async function loadHideoutData() {
    await Promise.all([
      ensureResource('armors', () => api.get<Armor[]>('/armors'), (data) => { armors.value = data }),
      ensureResource('armorInstances', () => api.get<ArmorInstance[]>('/armor-instances'), (data) => { armorInstances.value = data }),
      ensureResource('itemInstances', () => api.get<ItemInstance[]>('/item-instances'), (data) => { itemInstances.value = data }),
      ensureResource('consumables', () => api.get<Consumable[]>('/consumables'), (data) => { consumables.value = data }),
      ensureResource('storageCapacity', () => api.get<StorageCapacity>('/inventory/capacity'), (data) => { storageCapacity.value = data }),
      ensureResource('hideout', () => api.get<HideoutSnapshot>('/hideout'), (data) => { hideout.value = data }),
      ensureResource('craftingRecipes', () => api.get<CraftingRecipe[]>('/crafting/recipes'), (data) => { craftingRecipes.value = data }),
    ])
  }

  async function loadLogsData() {
    await Promise.all([
      ensureResource('maps', () => api.get<GameMap[]>('/maps'), (data) => { maps.value = data }),
      ensureResource('weapons', () => api.get<Weapon[]>('/weapons'), (data) => { weapons.value = data }),
    ])
  }

  async function loadViewData(view: NavKey): Promise<void> {
    const state = viewState[view]
    if (state.ready) return
    const pending = viewPromises.get(view)
    if (pending) return pending

    const generation = workspaceGeneration
    state.loading = true
    state.error = ''
    const task = (async () => {
      try {
        await loadCoreData()
        if (generation !== workspaceGeneration) return
        switch (view) {
          case 'explore': await loadExploreData(); break
          case 'live': await loadLiveData(); break
          case 'map': await loadMapData(); break
          case 'character': await loadCharacterData(); break
          case 'inventory': await loadInventoryData(); break
          case 'merchant': await loadMerchantData(); break
          case 'hideout': await loadHideoutData(); break
          case 'logs': await loadLogsData(); break
        }
        if (generation === workspaceGeneration) state.ready = true
      } catch (error) {
        if (generation === workspaceGeneration) state.error = getApiError(error, `${viewTitles[view]}加载失败`)
      } finally {
        if (generation === workspaceGeneration) state.loading = false
      }
    })()
    viewPromises.set(view, task)
    const cleanup = () => {
      if (viewPromises.get(view) === task) viewPromises.delete(view)
    }
    void task.then(cleanup, cleanup)
    return task
  }

  async function loadAll() {
    loading.value = true
    loadError.value = ''
    resetWorkspaceData()
    const generation = workspaceGeneration
    try {
      await loadCoreData()
    } catch (error) {
      if (generation !== workspaceGeneration) return
      loadError.value = getApiError(error, '玩家基础数据载入失败')
      loading.value = false
      return
    }
    if (generation !== workspaceGeneration) return
    loading.value = false
    await loadViewData(activeView.value)
  }
  
  async function refreshSessions() {
    try {
      const [sessionRes, playerRes, inventoryRes, storageCapacityRes, hideoutRes, recoveryRes, loadoutRes, armorInstancesRes, itemInstancesRes] = await Promise.all([
        api.get<Session[]>('/sessions'), api.get<Player>('/player'), api.get<InventoryItem[]>('/inventory'),
        api.get<StorageCapacity>('/inventory/capacity'),
        api.get<HideoutSnapshot>('/hideout'),
        api.get<RecoveryView | null>('/recovery/current'),
        api.get<PlayerLoadout>('/loadout'), api.get<ArmorInstance[]>('/armor-instances'), api.get<ItemInstance[]>('/item-instances'),
      ])
      sessions.value = sessionRes.data
      const nextActiveSession = sessions.value.find((item) => item.status === 'running')
      activeSessionId.value = nextActiveSession?.id ?? null
      player.value = playerRes.data
      inventory.value = inventoryRes.data
      storageCapacity.value = storageCapacityRes.data
      hideout.value = hideoutRes.data
      recovery.value = recoveryRes.data
      scheduleRecoveryPoll()
      loadout.value = loadoutRes.data
      armorInstances.value = armorInstancesRes.data
      itemInstances.value = itemInstancesRes.data
      markResourcesLoaded('sessions', 'player', 'inventory', 'storageCapacity', 'hideout', 'recovery', 'loadout', 'armorInstances', 'itemInstances')
    } catch (error) {
      ElMessage.error(getApiError(error, '行动状态刷新失败'))
    }
  }
  
  function saveLoadout(request: SaveLoadoutRequest, silent = false): Promise<void> {
    const generation = workspaceGeneration
    const version = ++loadoutSaveVersion
    pendingLoadoutSaves += 1
    savingLoadout.value = true

    const task = loadoutSaveQueue.then(async () => {
      try {
        if (workspaceStopped || generation !== workspaceGeneration) return

        const { data } = await api.put<PlayerLoadout>('/loadout', request)
        if (workspaceStopped || generation !== workspaceGeneration || version !== loadoutSaveVersion) return

        loadout.value = data
        resourceLoaded.loadout = true
        if (!silent) ElMessage.success('装备配置已保存')
      } catch (error) {
        if (workspaceStopped || generation !== workspaceGeneration || version !== loadoutSaveVersion) return

        try {
          const { data } = await api.get<PlayerLoadout>('/loadout')
          if (workspaceStopped || generation !== workspaceGeneration || version !== loadoutSaveVersion) return
          loadout.value = data
        } catch {
          // 保存失败后的快照同步失败时，保留原始保存错误，避免覆盖真正原因。
        }
        ElMessage.error(getApiError(error, '装备配置保存失败'))
      } finally {
        pendingLoadoutSaves -= 1
        savingLoadout.value = pendingLoadoutSaves > 0
      }
    })
    loadoutSaveQueue = task.catch(() => undefined)
    return task
  }
  
  async function purchaseItem(merchantId: string, itemId: string, quantity: number) {
    purchasingId.value = itemId
    try {
      await api.post('/merchant/purchase', { merchantId, itemId, quantity }, { headers: { 'Idempotency-Key': crypto.randomUUID() } })
      const [inventoryRes, capacityRes, armorInstancesRes, itemInstancesRes, hideoutRes] = await Promise.all([
        api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'), api.get<ArmorInstance[]>('/armor-instances'), api.get<ItemInstance[]>('/item-instances'), api.get<HideoutSnapshot>('/hideout'),
      ])
      inventory.value = inventoryRes.data
      storageCapacity.value = capacityRes.data
      armorInstances.value = armorInstancesRes.data
      itemInstances.value = itemInstancesRes.data
      hideout.value = hideoutRes.data
      markResourcesLoaded('inventory', 'storageCapacity', 'armorInstances', 'itemInstances', 'hideout')
      ElMessage.success('商品已存入仓库')
    } catch (error) {
      ElMessage.error(getApiError(error, '购买失败'))
    } finally {
      purchasingId.value = null
    }
  }
  
  async function sellItem(merchantId: string, itemId: string, quantity: number) {
    sellingId.value = itemId
    try {
      const { data } = await api.post<{ total: number }>('/merchant/sell', { merchantId, itemId, quantity }, { headers: { 'Idempotency-Key': crypto.randomUUID() } })
      const [inventoryRes, capacityRes, itemInstancesRes, hideoutRes] = await Promise.all([
        api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'), api.get<ItemInstance[]>('/item-instances'), api.get<HideoutSnapshot>('/hideout'),
      ])
      inventory.value = inventoryRes.data
      storageCapacity.value = capacityRes.data
      itemInstances.value = itemInstancesRes.data
      hideout.value = hideoutRes.data
      markResourcesLoaded('inventory', 'storageCapacity', 'itemInstances', 'hideout')
      ElMessage.success(`已出售，获得 ￥${data.total}`)
    } catch (error) {
      ElMessage.error(getApiError(error, '出售失败'))
    } finally {
      sellingId.value = null
    }
  }
  
  async function savePlayerName(name: string) {
    savingPlayer.value = true
    try {
      const { data } = await api.put<Player>('/player', { name })
      player.value = data
      resourceLoaded.player = true
      ElMessage.success('角色名称已保存')
    } catch (error) {
      ElMessage.error(getApiError(error, '角色名称保存失败'))
    } finally {
      savingPlayer.value = false
    }
  }
  
  async function repairArmor(id: number) {
    repairingId.value = id
    try {
      await api.post('/hideout/repair', { armorInstanceId: id })
      const [armorInstancesRes, hideoutRes, capacityRes] = await Promise.all([
        api.get<ArmorInstance[]>('/armor-instances'), api.get<HideoutSnapshot>('/hideout'), api.get<StorageCapacity>('/inventory/capacity'),
      ])
      armorInstances.value = armorInstancesRes.data
      hideout.value = hideoutRes.data
      storageCapacity.value = capacityRes.data
      markResourcesLoaded('armorInstances', 'hideout', 'storageCapacity')
      ElMessage.success('护甲已加入维修队列')
    } catch (error) {
      ElMessage.error(getApiError(error, '护甲维修失败'))
    } finally {
      repairingId.value = null
    }
  }

  async function upgradeFacility(facilityId: string) {
    upgradingFacilityId.value = facilityId
    try {
      await api.post(`/hideout/facilities/${facilityId}/upgrade`)
      const [inventoryRes, capacityRes, hideoutRes] = await Promise.all([
        api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'), api.get<HideoutSnapshot>('/hideout'),
      ])
      inventory.value = inventoryRes.data
      storageCapacity.value = capacityRes.data
      hideout.value = hideoutRes.data
      markResourcesLoaded('inventory', 'storageCapacity', 'hideout')
      ElMessage.success('设施升级已开始')
    } catch (error) {
      ElMessage.error(getApiError(error, '设施升级失败'))
    } finally {
      upgradingFacilityId.value = null
    }
  }

  async function refreshHideoutResources() {
    const [hideoutRes, itemInstancesRes, inventoryRes, capacityRes] = await Promise.all([
      api.get<HideoutSnapshot>('/hideout'), api.get<ItemInstance[]>('/item-instances'), api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'),
    ])
    hideout.value = hideoutRes.data
    itemInstances.value = itemInstancesRes.data
    inventory.value = inventoryRes.data
    storageCapacity.value = capacityRes.data
    markResourcesLoaded('hideout', 'itemInstances', 'inventory', 'storageCapacity')
  }

  async function refreshCraftingState() {
    const [recipesRes, hideoutRes, inventoryRes, capacityRes, itemInstancesRes] = await Promise.all([
      api.get<CraftingRecipe[]>('/crafting/recipes'),
      api.get<HideoutSnapshot>('/hideout'),
      api.get<InventoryItem[]>('/inventory'),
      api.get<StorageCapacity>('/inventory/capacity'),
      api.get<ItemInstance[]>('/item-instances'),
    ])
    craftingRecipes.value = recipesRes.data
    hideout.value = hideoutRes.data
    inventory.value = inventoryRes.data
    storageCapacity.value = capacityRes.data
    itemInstances.value = itemInstancesRes.data
    markResourcesLoaded('craftingRecipes', 'hideout', 'inventory', 'storageCapacity', 'itemInstances')
  }

  async function startCraft(recipeId: string) {
    craftingId.value = recipeId
    try {
      await api.post('/crafting/start', { recipeId })
      await refreshCraftingState()
      ElMessage.success('制造已开始')
    } catch (error) {
      ElMessage.error(getApiError(error, '制造失败'))
    } finally {
      craftingId.value = null
    }
  }

  async function toggleGenerator(enabled: boolean) {
    try {
      await api.post('/hideout/generator/toggle', { enabled })
      await refreshHideoutResources()
      ElMessage.success(enabled ? '发电机已启动' : '发电机已关闭')
    } catch (error) {
      ElMessage.error(getApiError(error, '发电机状态更新失败'))
    }
  }

  async function loadGeneratorFuel(instanceId: number) {
    try {
      await api.post('/hideout/generator/fuel/load', null, { params: { instanceId } })
      await refreshHideoutResources()
      ElMessage.success('燃料已装入发电机')
    } catch (error) {
      ElMessage.error(getApiError(error, '装载燃料失败'))
    }
  }

  async function unloadGeneratorFuel(instanceId: number) {
    try {
      await api.post('/hideout/generator/fuel/unload', null, { params: { instanceId } })
      await refreshHideoutResources()
      ElMessage.success('燃料已取回仓库')
    } catch (error) {
      ElMessage.error(getApiError(error, '卸载燃料失败'))
    }
  }
  
  async function handleSessionCreated(session: Session) {
    await refreshSessions()
    activeSessionId.value = session.id
    activeView.value = 'live'
  }
  
  async function initialize() {
    try {
      const { data } = await api.get<User>('/auth/me')
      user.value = data
    } catch (error) {
      if (!isUnauthorized(error)) authError.value = getApiError(error, '服务器连接失败，请稍后重试')
      user.value = null
      authChecking.value = false
      return
    }
    authChecking.value = false
    await loadAll()
  }
  
  async function handleAuthenticated(nextUser: User) {
    user.value = nextUser
    authError.value = ''
    await loadAll()
  }
  
  async function logout() {
    try {
      await api.post('/auth/logout')
      resetWorkspaceData()
      user.value = null
    } catch (error) {
      ElMessage.error(getApiError(error, '退出登录失败'))
    }
  }

  watch(activeView, (view) => {
    if (user.value && !loading.value) void loadViewData(view)
  }, { flush: 'sync' })

  onMounted(initialize)

  onUnmounted(() => {
    workspaceStopped = true
    clearRecoveryPoll()
  })

  return {
    activeView, user, authChecking, authError, mobileOpen, loading, loadError, viewState,
    savingPlayer, savingLoadout, purchasingId, sellingId, repairingId, upgradingFacilityId, craftingId,
    player, loadout, maps, mapGraphs, enemies, weapons, ammos, armors, armorInstances, itemInstances,
    consumables, chestRigs, backpacks, helmets, headsets, merchants, inventory,
    storageCapacity, hideout, craftingRecipes, recovery, sessions, activeSessionId, viewTitles, cash, latestSession, activeSession,
    loadAll, loadViewData, refreshSessions, saveLoadout, purchaseItem, sellItem, savePlayerName,
    repairArmor, upgradeFacility, toggleGenerator, loadGeneratorFuel, unloadGeneratorFuel, startCraft, handleSessionCreated, handleAuthenticated, logout,
  }
}
