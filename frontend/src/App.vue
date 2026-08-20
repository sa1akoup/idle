<!-- 应用工作台：加载全局游戏数据并组织七个功能视图。 -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Menu, Refresh } from '@element-plus/icons-vue'
import api, { getApiError } from './api'
import AppSidebar from './components/AppSidebar.vue'
import CharacterView from './views/CharacterView.vue'
import ExploreView from './views/ExploreView.vue'
import HideoutView from './views/HideoutView.vue'
import InventoryView from './views/InventoryView.vue'
import LogsView from './views/LogsView.vue'
import MapView from './views/MapView.vue'
import MerchantView from './views/MerchantView.vue'
import type {
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
  MapNode,
  Merchant,
  NavKey,
  Player,
  PlayerLoadout,
  SaveLoadoutRequest,
  Session,
  StorageCapacity,
  Weapon,
} from './types'

const activeView = ref<NavKey>('explore')
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
const nodes = ref<MapNode[]>([])
const enemies = ref<Enemy[]>([])
const weapons = ref<Weapon[]>([])
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

const viewTitles: Record<NavKey, string> = {
  explore: '探索部署', map: '区域地图', character: '玩家角色', inventory: '本地仓库',
  merchant: '灰区商人', hideout: '藏身处', logs: '行动日志',
}

const cash = computed(() => inventory.value.find((item) => item.itemId === 'cash')?.quantity ?? 0)
const latestSession = computed(() => sessions.value[0] ?? null)

async function loadAll() {
  loading.value = true
  loadError.value = ''
  try {
    const [
      playerRes, mapsRes, nodesRes, enemiesRes, weaponsRes, armorsRes,
      armorInstancesRes, consumablesRes, chestRigsRes, backpacksRes, helmetsRes, headsetsRes,
      inventoryRes, storageCapacityRes, sessionsRes, loadoutRes, merchantsRes,
    ] = await Promise.all([
      api.get<Player>('/player'), api.get<GameMap[]>('/maps'), api.get<MapNode[]>('/nodes'),
      api.get<Enemy[]>('/enemies'), api.get<Weapon[]>('/weapons'), api.get<Armor[]>('/armors'),
      api.get<ArmorInstance[]>('/armor-instances'), api.get<Consumable[]>('/consumables'),
      api.get<ChestRig[]>('/chestrigs'), api.get<Backpack[]>('/backpacks'), api.get<Helmet[]>('/helmets'), api.get<Headset[]>('/headsets'),
      api.get<InventoryItem[]>('/inventory'),
      api.get<StorageCapacity>('/inventory/capacity'),
      api.get<Session[]>('/sessions'), api.get<PlayerLoadout>('/loadout'),
      api.get<Merchant[]>('/merchants'),
    ])
    player.value = playerRes.data
    maps.value = mapsRes.data
    nodes.value = nodesRes.data
    enemies.value = enemiesRes.data
    weapons.value = weaponsRes.data
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

async function purchaseItem(merchantId: string, itemId: string) {
  purchasingId.value = itemId
  try {
    await api.post('/merchant/purchase', { merchantId, itemId, quantity: 1 })
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
    const { data } = await api.post<{ total: number }>('/merchant/sell', { merchantId, itemId, quantity })
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

async function handleSessionCreated() {
  await refreshSessions()
  activeView.value = 'logs'
}

onMounted(loadAll)
</script>

<template>
  <div class="app-shell">
    <AppSidebar
      :active="activeView"
      :mobile-open="mobileOpen"
      :player="player"
      :cash="cash"
      @select="activeView = $event"
      @close="mobileOpen = false"
    />

    <main class="app-main">
      <header class="topbar">
        <button type="button" class="mobile-menu" title="打开功能菜单" @click="mobileOpen = true"><el-icon><Menu /></el-icon></button>
        <div><span>行动终端</span><strong>{{ viewTitles[activeView] }}</strong></div>
        <div class="topbar-status">
          <span><i class="status-dot" />服务器在线</span>
          <span v-if="latestSession">最近行动 #{{ latestSession.id }} · {{ latestSession.totalRuns }} 局</span>
        </div>
      </header>

      <div class="app-content">
        <div v-if="loading" class="loading-shell"><el-skeleton :rows="8" animated /></div>
        <div v-else-if="loadError" class="fatal-state">
          <strong>行动终端无法载入</strong><p>{{ loadError }}</p><el-button :icon="Refresh" @click="loadAll">重新连接</el-button>
        </div>
        <template v-else-if="player">
          <ExploreView
            v-if="activeView === 'explore' && loadout"
            :player="player" :loadout="loadout" :maps="maps" :weapons="weapons" :armors="armors" :consumables="consumables"
            @created="handleSessionCreated"
          />
          <MapView v-else-if="activeView === 'map'" :maps="maps" :nodes="nodes" :enemies="enemies" />
          <CharacterView
            v-else-if="activeView === 'character' && loadout"
            :player="player" :loadout="loadout" :inventory="inventory" :weapons="weapons" :armors="armors" :consumables="consumables"
            :chest-rigs="chestRigs" :backpacks="backpacks" :helmets="helmets" :headsets="headsets"
            :saving-name="savingPlayer" :saving-loadout="savingLoadout"
            @save-name="savePlayerName" @save-loadout="saveLoadout"
          />
          <InventoryView v-else-if="activeView === 'inventory'" :inventory="inventory" :loadout="loadout" :storage-capacity="storageCapacity" />
          <MerchantView
            v-else-if="activeView === 'merchant'"
            :merchants="merchants" :inventory="inventory"
            :purchasing-id="purchasingId" :selling-id="sellingId"
            @purchase="purchaseItem" @sell="sellItem"
          />
          <HideoutView
            v-else-if="activeView === 'hideout'"
            :player="player" :armors="armors" :armor-instances="armorInstances" :repairing-id="repairingId" :storage-capacity="storageCapacity"
            @repair="repairArmor"
          />
          <LogsView v-else :sessions="sessions" :maps="maps" :weapons="weapons" @refresh="refreshSessions" />
        </template>
      </div>
    </main>
  </div>
</template>
