<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Refresh, Sell } from '@element-plus/icons-vue'
import type { InventoryItem, ItemInstance, Merchant, MerchantCatalogItem } from '../types'
import { formatMerchantRefresh, useMerchantCatalog } from '../composables/useMerchantCatalog'

const props = defineProps<{
  merchants: Merchant[]
  inventory: InventoryItem[]
  itemInstances: ItemInstance[]
  purchasingId: string | null
  sellingId: string | null
}>()

const emit = defineEmits<{ purchase: [merchantId: string, itemId: string, quantity: number]; sell: [merchantId: string, itemId: string, quantity: number] }>()

const {
  selectedId, catalogLoading, catalogError, acceptsAny, nextRefreshAt, sellQty, buyQty,
  selectedMerchant, buyableCatalog, sellable, loadCatalog, qtyFor, ownedQty, buyQuantity,
} = useMerchantCatalog(toRef(props, 'merchants'), toRef(props, 'inventory'), toRef(props, 'itemInstances'), toRef(props, 'purchasingId'))

const cash = computed(() => props.inventory.find((item) => item.itemId === 'cash')?.quantity ?? 0)

function isBarter(item: MerchantCatalogItem): boolean {
  return (item.barterCosts?.length ?? 0) > 0
}

function canBuy(item: MerchantCatalogItem): boolean {
  if (!selectedMerchant.value || props.purchasingId !== null) return false
  if (selectedMerchant.value.reputation < item.repRequirement) return false
  if (isBarter(item)) {
    if (item.barterLocked) return false
    return (item.barterCosts ?? []).every((cost) => cost.have >= cost.need)
  }
  const quantity = buyQuantity(item)
  return cash.value >= item.price * quantity && quantity <= (item.stock ?? 999)
}
</script>

<template>
  <section class="view-page merchant-view">
    <header class="page-heading">
      <div><span class="eyebrow">灰区交易频道</span><h1>商人</h1><p>专精商人只买卖本类；黑市收购任意货物，货架每 6 小时按稀有度刷新且售价翻倍。</p></div>
      <div class="channel-state"><span class="status-dot online" />开放交易</div>
    </header>

    <div class="merchant-grid">
      <button
        v-for="merchant in merchants" :key="merchant.id" type="button"
        class="merchant-card" :class="{ active: selectedId === merchant.id, closed: !merchant.open }"
        @click="loadCatalog(merchant.id)"
      >
        <div class="merchant-card__head"><strong>{{ merchant.name }}</strong><span :class="merchant.open ? 'open' : 'closed'">{{ merchant.open ? '营业' : '待开放' }}</span></div>
        <p>{{ merchant.desc }}</p>
        <div class="merchant-card__rep">好感度 <b>{{ merchant.reputation }}</b></div>
      </button>
    </div>

    <div v-if="selectedMerchant" class="merchant-detail">
      <div class="merchant-detail__title">
        <div><strong>{{ selectedMerchant.name }}</strong><span>{{ selectedMerchant.desc }}</span></div>
        <b>好感度 {{ selectedMerchant.reputation }} <small>影响买卖价格</small></b>
      </div>

      <div v-if="!selectedMerchant.open" class="merchant-closed">
        <p>该商人暂未开放</p><span>提升商人好感度后可解锁交易</span>
      </div>

      <template v-else>
        <div class="merchant-summary">
          <span>现金 ￥{{ cash.toLocaleString() }}</span>
          <span>{{ buyableCatalog.length }} 件可购买</span>
          <span v-if="nextRefreshAt">下次刷新 {{ formatMerchantRefresh(nextRefreshAt) }}</span>
        </div>

        <section class="catalog-block">
          <div class="panel-heading"><div><span>SHOP</span><h2>出售给玩家</h2></div><el-button :icon="Refresh" :loading="catalogLoading" circle title="刷新目录" @click="loadCatalog(selectedId)" /></div>
          <div v-if="catalogError" class="text-danger">{{ catalogError }}</div>
          <div v-else class="catalog-list surface-panel">
            <div v-for="item in buyableCatalog" :key="item.id" class="catalog-row">
              <div><small>{{ item.kind }}</small><strong>{{ item.name }}</strong><p>{{ item.detail }} · {{ item.weight }}kg/{{ item.slots }}格 · 已有 {{ ownedQty(item.id) }}<template v-if="item.stock !== undefined"> · 库存 {{ item.stock }}</template></p></div>
              <div class="catalog-action">
                <div v-if="isBarter(item)" class="barter-costs">
                  <small v-for="cost in item.barterCosts" :key="cost.itemId" :class="{ ready: cost.have >= cost.need }">{{ cost.name }} {{ cost.have }}/{{ cost.need }}</small>
                </div>
                <b v-else>￥{{ item.price.toLocaleString() }}</b>
                <small v-if="item.barterLockReason">{{ item.barterLockReason }}</small>
                <small v-if="item.repRequirement > 0">需好感度 {{ item.repRequirement }}</small>
                <el-input-number v-if="!isBarter(item) && (item.kind === 'ammo' || (item.stock !== undefined && item.stock > 1))" v-model="buyQty[item.id]" :min="1" :max="item.stock ?? 999" :step="item.kind === 'ammo' ? 30 : 1" size="small" />
                <el-button type="primary" size="small" :loading="purchasingId === item.id" :disabled="!canBuy(item)" @click="emit('purchase', selectedId, item.id, isBarter(item) ? 1 : buyQuantity(item))">{{ isBarter(item) ? '兑换' : '购买' }}</el-button>
              </div>
            </div>
            <el-empty v-if="buyableCatalog.length === 0" description="暂无在售商品" />
          </div>
        </section>

        <section class="catalog-block">
          <div class="panel-heading"><div><span>BUYBACK</span><h2>{{ acceptsAny ? '收购任意货物' : '收购玩家' }}</h2></div></div>
          <div class="catalog-list surface-panel">
            <div v-for="item in sellable" :key="item.itemId" class="catalog-row">
              <div><small>可出售</small><strong>{{ item.name }}</strong><p>持有 {{ item.quantity }} · 单价 ￥{{ item.sellPrice.toLocaleString() }}</p></div>
              <div class="catalog-action">
                <el-input-number v-model="sellQty[item.itemId]" :min="1" :max="item.quantity" size="small" />
                <el-button type="primary" size="small" :icon="Sell" :loading="sellingId === item.itemId" :disabled="sellingId !== null" @click="emit('sell', selectedId, item.itemId, qtyFor(item.itemId))">出售</el-button>
              </div>
            </div>
            <el-empty v-if="sellable.length === 0" description="暂无同类物品可出售" />
          </div>
        </section>
      </template>
    </div>
  </section>
</template>
