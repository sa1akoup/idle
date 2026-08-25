// 应用工作区状态：集中加载全局数据、处理用户操作并维护功能视图。
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api, { getApiError, isUnauthorized, setUnauthorizedHandler } from '../api'
import type {
  Ammo,
  Armor,
  ArmorInstance,
  Backpack,
  ChestRig,
  Consumable,
  Enemy,
  GameMap,
  Headset,
  Helmet,
  InventoryItem,
  MapGraph,
  MapNode,
  Merchant,
  NavKey,
  Player,
  PlayerLoadout,
  SaveLoadoutRequest,
  Session,
  StorageCapacity,
  User,
  Weapon,
} from '../types'

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
  
  const player = ref<Player | null>(null)
  const loadout = ref<PlayerLoadout | null>(null)
  const maps = ref<GameMap[]>([])
  const mapGraphs = ref<Record<string, MapGraph>>({})
  const nodes = ref<MapNode[]>([])
  const enemies = ref<Enemy[]>([])
  const weapons = ref<Weapon[]>([])
  const ammos = ref<Ammo[]>([])
  const armors = ref<Armor[]>([])
  const armorInstances = ref<ArmorInstance[]>([])
  const consumables = ref<Consumable[]>([])
  const chestRigs = ref<ChestRig[]>([])
  const backpacks = ref<Backpack[]>([])
  const helmets = ref<Helmet[]>([])
  const headsets = ref<Headset[]>([])
  const merchants = ref<Merchant[]>([])
  const inventory = ref<InventoryItem[]>([])
  const storageCapacity = ref<StorageCapacity | null>(null)
  const sessions = ref<Session[]>([])
  const activeSessionId = ref<number | null>(null)
  
  const viewTitles: Record<NavKey, string> = {
    explore: '探索部署', live: '实时行动', map: '区域地图', character: '玩家角色', inventory: '本地仓库',
    merchant: '灰区商人', hideout: '藏身处', logs: '行动日志',
  }
  
  const cash = computed(() => inventory.value.find((item) => item.itemId === 'cash')?.quantity ?? 0)
  const latestSession = computed(() => sessions.value[0] ?? null)
  const activeSession = computed(() => sessions.value.find((item) => item.status === 'running' || item.status === 'waiting_injury') ?? null)
  
  setUnauthorizedHandler(() => {
    user.value = null
    player.value = null
    loadout.value = null
  })
  
  async function loadAll() {
    loading.value = true
    loadError.value = ''
    try {
      const [
        playerRes, mapsRes, enemiesRes, weaponsRes, ammosRes, armorsRes,
        armorInstancesRes, consumablesRes, chestRigsRes, backpacksRes, helmetsRes, headsetsRes,
        inventoryRes, storageCapacityRes, sessionsRes, loadoutRes, merchantsRes,
      ] = await Promise.all([
        api.get<Player>('/player'), api.get<GameMap[]>('/maps'),
        api.get<Enemy[]>('/enemies'), api.get<Weapon[]>('/weapons'), api.get<Ammo[]>('/ammos'), api.get<Armor[]>('/armors'),
        api.get<ArmorInstance[]>('/armor-instances'), api.get<Consumable[]>('/consumables'),
        api.get<ChestRig[]>('/chestrigs'), api.get<Backpack[]>('/backpacks'), api.get<Helmet[]>('/helmets'), api.get<Headset[]>('/headsets'),
        api.get<InventoryItem[]>('/inventory'),
        api.get<StorageCapacity>('/inventory/capacity'),
        api.get<Session[]>('/sessions'), api.get<PlayerLoadout>('/loadout'),
        api.get<Merchant[]>('/merchants'),
      ])
      player.value = playerRes.data
      maps.value = mapsRes.data
      const graphResults = await Promise.all(mapsRes.data.map((map) => api.get<MapGraph>(`/maps/${map.id}/graph`)))
      mapGraphs.value = Object.fromEntries(graphResults.map((response) => [response.data.map.id, response.data]))
      nodes.value = graphResults.flatMap((response) => response.data.nodes)
      enemies.value = enemiesRes.data
      weapons.value = weaponsRes.data
      ammos.value = ammosRes.data
      armors.value = armorsRes.data
      armorInstances.value = armorInstancesRes.data
      consumables.value = consumablesRes.data
      chestRigs.value = chestRigsRes.data
      backpacks.value = backpacksRes.data
      helmets.value = helmetsRes.data
      headsets.value = headsetsRes.data
      inventory.value = inventoryRes.data
      storageCapacity.value = storageCapacityRes.data
      sessions.value = sessionsRes.data
      activeSessionId.value = sessions.value.find((item) => item.status === 'running' || item.status === 'waiting_injury')?.id ?? null
      loadout.value = loadoutRes.data
      merchants.value = merchantsRes.data
    } catch (error) {
      loadError.value = getApiError(error, '游戏数据载入失败')
    } finally {
      loading.value = false
    }
  }
  
  async function refreshSessions() {
    try {
      const [sessionRes, playerRes, inventoryRes, storageCapacityRes, loadoutRes, armorInstancesRes] = await Promise.all([
        api.get<Session[]>('/sessions'), api.get<Player>('/player'), api.get<InventoryItem[]>('/inventory'),
        api.get<StorageCapacity>('/inventory/capacity'),
        api.get<PlayerLoadout>('/loadout'), api.get<ArmorInstance[]>('/armor-instances'),
      ])
      sessions.value = sessionRes.data
      const nextActiveSession = sessions.value.find((item) => item.status === 'running' || item.status === 'waiting_injury')
      if (nextActiveSession) activeSessionId.value = nextActiveSession.id
      player.value = playerRes.data
      inventory.value = inventoryRes.data
      storageCapacity.value = storageCapacityRes.data
      loadout.value = loadoutRes.data
      armorInstances.value = armorInstancesRes.data
    } catch (error) {
      ElMessage.error(getApiError(error, '行动状态刷新失败'))
    }
  }
  
  async function saveLoadout(request: SaveLoadoutRequest, silent = false) {
    savingLoadout.value = true
    try {
      const { data } = await api.put<PlayerLoadout>('/loadout', request)
      loadout.value = data
      if (!silent) ElMessage.success('装备配置已保存')
    } catch (error) {
      ElMessage.error(getApiError(error, '装备配置保存失败'))
    } finally {
      savingLoadout.value = false
    }
  }
  
  async function purchaseItem(merchantId: string, itemId: string, quantity: number) {
    purchasingId.value = itemId
    try {
      await api.post('/merchant/purchase', { merchantId, itemId, quantity }, { headers: { 'Idempotency-Key': crypto.randomUUID() } })
      const [inventoryRes, capacityRes, armorInstancesRes] = await Promise.all([
        api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'), api.get<ArmorInstance[]>('/armor-instances'),
      ])
      inventory.value = inventoryRes.data
      storageCapacity.value = capacityRes.data
      armorInstances.value = armorInstancesRes.data
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
      const [inventoryRes, capacityRes] = await Promise.all([
        api.get<InventoryItem[]>('/inventory'), api.get<StorageCapacity>('/inventory/capacity'),
      ])
      inventory.value = inventoryRes.data
      storageCapacity.value = capacityRes.data
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
      await api.post('/armor/repair', { id })
      const { data } = await api.get<ArmorInstance[]>('/armor-instances')
      armorInstances.value = data
      ElMessage.success('护甲翻修完成')
    } catch (error) {
      ElMessage.error(getApiError(error, '护甲维修失败'))
    } finally {
      repairingId.value = null
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
      user.value = null
      player.value = null
    } catch (error) {
      ElMessage.error(getApiError(error, '退出登录失败'))
    }
  }
  
  onMounted(initialize)

  return {
    activeView, user, authChecking, authError, mobileOpen, loading, loadError,
    savingPlayer, savingLoadout, purchasingId, sellingId, repairingId,
    player, loadout, maps, mapGraphs, nodes, enemies, weapons, ammos, armors, armorInstances,
    consumables, chestRigs, backpacks, helmets, headsets, merchants, inventory,
    storageCapacity, sessions, activeSessionId, viewTitles, cash, latestSession, activeSession,
    loadAll, refreshSessions, saveLoadout, purchaseItem, sellItem, savePlayerName,
    repairArmor, handleSessionCreated, handleAuthenticated, logout,
  }
}
