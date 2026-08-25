<!-- 应用工作台：加载全局游戏数据并组织七个功能视图。 -->
<script setup lang="ts">
import { Menu, Refresh } from '@element-plus/icons-vue'
import AppSidebar from './components/AppSidebar.vue'
import AuthView from './views/AuthView.vue'
import CharacterView from './views/CharacterView.vue'
import ExploreView from './views/ExploreView.vue'
import HideoutView from './views/HideoutView.vue'
import InventoryView from './views/InventoryView.vue'
import LiveSessionView from './views/LiveSessionView.vue'
import LogsView from './views/LogsView.vue'
import MapView from './views/MapView.vue'
import MerchantView from './views/MerchantView.vue'
import { useAppWorkspace } from './composables/useAppWorkspace'

const {
  activeView, user, authChecking, authError, mobileOpen, loading, loadError,
  savingPlayer, savingLoadout, purchasingId, sellingId, repairingId,
  player, loadout, maps, mapGraphs, enemies, weapons, ammos, armors, armorInstances,
  consumables, chestRigs, backpacks, helmets, headsets, merchants, inventory,
  storageCapacity, sessions, activeSessionId, viewTitles, cash, latestSession, activeSession,
  loadAll, refreshSessions, saveLoadout, purchaseItem, sellItem, savePlayerName,
  repairArmor, handleSessionCreated, handleAuthenticated, logout,
} = useAppWorkspace()
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
          <button type="button" class="logout-button" @click="logout">退出</button>
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
            :player="player" :loadout="loadout" :maps="maps" :weapons="weapons" :ammos="ammos" :armors="armors" :consumables="consumables" :inventory="inventory"
            @created="handleSessionCreated"
          />
          <MapView v-else-if="activeView === 'map'" :maps="maps" :map-graphs="mapGraphs" :enemies="enemies" />
          <CharacterView
            v-else-if="activeView === 'character' && loadout"
            :player="player" :loadout="loadout" :inventory="inventory" :weapons="weapons" :ammos="ammos" :armors="armors" :consumables="consumables"
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
          <LiveSessionView
            v-else-if="activeView === 'live' && activeSessionId"
            :key="activeSessionId" :session-id="activeSessionId" :player="player" :maps="maps" :map-graphs="mapGraphs"
            @refresh="refreshSessions" @open-logs="activeView = 'logs'"
          />
          <LogsView v-else :sessions="sessions" :maps="maps" :weapons="weapons" @refresh="refreshSessions" />
        </template>
      </div>
    </main>
  </div>
</template>
