<!-- 应用工作台：加载全局游戏数据并组织七个功能视图。 -->
<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { Menu, Refresh } from '@element-plus/icons-vue'
import AppSidebar from './components/AppSidebar.vue'
import AuthView from './views/AuthView.vue'
import { useAppWorkspace } from './composables/useAppWorkspace'
import type { NavKey } from './types'

const CharacterView = defineAsyncComponent(() => import('./views/CharacterView.vue'))
const ExploreView = defineAsyncComponent(() => import('./views/ExploreView.vue'))
const HideoutView = defineAsyncComponent(() => import('./views/HideoutView.vue'))
const InventoryView = defineAsyncComponent(() => import('./views/InventoryView.vue'))
const LiveSessionView = defineAsyncComponent(() => import('./views/LiveSessionView.vue'))
const LogsView = defineAsyncComponent(() => import('./views/LogsView.vue'))
const MapView = defineAsyncComponent(() => import('./views/MapView.vue'))
const MerchantView = defineAsyncComponent(() => import('./views/MerchantView.vue'))

const {
  activeView, user, authChecking, authError, mobileOpen, loading, loadError,
  viewState,
  savingPlayer, savingLoadout, purchasingId, sellingId, repairingId,
  player, loadout, maps, mapGraphs, enemies, weapons, ammos, armors, armorInstances, itemInstances,
  consumables, chestRigs, backpacks, helmets, headsets, merchants, inventory,
  storageCapacity, hideout, craftingRecipes, recovery, sessions, activeSessionId, viewTitles, cash, latestSession, activeSession,
  loadAll, loadViewData, refreshSessions, saveLoadout, purchaseItem, sellItem, savePlayerName,
  repairArmor, upgradeFacility, toggleGenerator, loadGeneratorFuel, unloadGeneratorFuel, craftingId, upgradingFacilityId, startCraft, handleSessionCreated, handleAuthenticated, logout,
} = useAppWorkspace()

function selectView(view: NavKey) {
  activeView.value = view
  void loadViewData(view)
}
</script>

<template>
  <div v-if="authChecking" class="loading-shell"><el-skeleton :rows="8" animated /></div>
  <AuthView v-else-if="!user" :error="authError" @authenticated="handleAuthenticated" />
  <div v-else class="app-shell">
    <AppSidebar
      :active="activeView"
      :mobile-open="mobileOpen"
      :player="player"
      :cash="cash"
      :has-active-session="Boolean(activeSession)"
      @select="selectView"
      @close="mobileOpen = false"
    />

    <main class="app-main">
      <header class="topbar">
        <button type="button" class="mobile-menu" title="打开功能菜单" @click="mobileOpen = true"><el-icon><Menu /></el-icon></button>
        <div><span>行动终端</span><strong>{{ viewTitles[activeView] }}</strong></div>
        <div class="topbar-status">
          <span><i class="status-dot" />服务器在线</span>
          <span v-if="latestSession">最近行动 #{{ latestSession.id }} · {{ latestSession.totalRuns }} 局</span>
          <button type="button" class="logout-button" @click="logout">退出</button>
        </div>
      </header>

      <div class="app-content">
        <div v-if="loading" class="loading-shell"><el-skeleton :rows="8" animated /></div>
        <div v-else-if="loadError" class="fatal-state">
          <strong>行动终端无法载入</strong><p>{{ loadError }}</p><el-button :icon="Refresh" @click="loadAll">重新连接</el-button>
        </div>
        <template v-else-if="player">
          <div v-if="viewState[activeView].loading" class="loading-shell"><el-skeleton :rows="8" animated /></div>
          <div v-else-if="viewState[activeView].error" class="fatal-state">
            <strong>{{ viewTitles[activeView] }}暂不可用</strong>
            <p>{{ viewState[activeView].error }}</p>
            <el-button :icon="Refresh" @click="loadViewData(activeView)">重新载入</el-button>
          </div>
          <template v-else>
            <ExploreView
              v-if="activeView === 'explore' && loadout"
              :player="player" :loadout="loadout" :maps="maps" :weapons="weapons" :ammos="ammos" :armors="armors" :consumables="consumables" :inventory="inventory" :recovery="recovery"
              @created="handleSessionCreated"
            />
            <MapView v-else-if="activeView === 'map'" :maps="maps" :map-graphs="mapGraphs" :enemies="enemies" />
            <CharacterView
              v-else-if="activeView === 'character' && loadout"
              :player="player" :loadout="loadout" :inventory="inventory" :item-instances="itemInstances" :weapons="weapons" :ammos="ammos" :armors="armors" :consumables="consumables"
              :chest-rigs="chestRigs" :backpacks="backpacks" :helmets="helmets" :headsets="headsets" :merchants="merchants"
              :saving-name="savingPlayer" :saving-loadout="savingLoadout"
              @save-name="savePlayerName" @save-loadout="saveLoadout"
            />
            <InventoryView v-else-if="activeView === 'inventory'" :inventory="inventory" :item-instances="itemInstances" :loadout="loadout" :storage-capacity="storageCapacity" />
            <MerchantView
              v-else-if="activeView === 'merchant'"
              :merchants="merchants" :inventory="inventory" :item-instances="itemInstances"
              :purchasing-id="purchasingId" :selling-id="sellingId"
              @purchase="purchaseItem" @sell="sellItem"
            />
            <HideoutView
              v-else-if="activeView === 'hideout'"
              :player="player" :armors="armors" :armor-instances="armorInstances" :item-instances="itemInstances" :consumables="consumables" :repairing-id="repairingId" :storage-capacity="storageCapacity"
              :hideout="hideout" :upgrading-facility-id="upgradingFacilityId" :crafting-recipes="craftingRecipes" :crafting-id="craftingId"
              @repair="repairArmor" @upgrade="upgradeFacility" @toggle-generator="toggleGenerator" @load-generator-fuel="loadGeneratorFuel" @unload-generator-fuel="unloadGeneratorFuel" @craft="startCraft"
            />
            <LiveSessionView
              v-else-if="activeView === 'live' && activeSessionId"
              :key="activeSessionId" :session-id="activeSessionId" :player="player" :maps="maps" :map-graphs="mapGraphs"
              @refresh="refreshSessions" @open-logs="selectView('logs')"
            />
            <LogsView v-else :sessions="sessions" :maps="maps" :weapons="weapons" />
          </template>
        </template>
      </div>
    </main>
  </div>
</template>
