<!-- 商人页：6 类商人，按类别买卖对应物品，好感度影响价格并解锁高级商品。 -->
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Refresh, Sell } from '@element-plus/icons-vue'
import api, { getApiError } from '../api'
import type { InventoryItem, Merchant, MerchantCatalogItem } from '../types'

const props = defineProps<{
  merchants: Merchant[]
  inventory: InventoryItem[]
  purchasingId: string | null
  sellingId: string | null
}>()

const emit = defineEmits<{ purchase: [merchantId: string, itemId: string]; sell: [merchantId: string, itemId: string, quantity: number] }>()

const selectedId = ref('')
const catalog = ref<MerchantCatalogItem[]>([])
const catalogLoading = ref(false)
const catalogError = ref('')
const sellQty = ref<Record<string, number>>({})

const selectedMerchant = computed(() => props.merchants.find((m) => m.id === selectedId.value))

const cash = computed(() => props.inventory.find((i) => i.itemId === 'cash')?.quantity ?? 0)

async function loadCatalog(id: string) {
  selectedId.value = id
  catalog.value = []
  catalogError.value = ''
  const m = props.merchants.find((x) => x.id === id)
  if (!m?.open) return
  catalogLoading.value = true
  try {
    const { data } = await api.get<MerchantCatalogItem[]>(`/merchants/${id}/catalog`)
    catalog.value = data
  } catch (error) {
    catalogError.value = getApiError(error, '商品目录加载失败')
  } finally {
    catalogLoading.value = false
  }
}

// 可出售给当前商人的同类别物品（不限来源，聚合两行库存数量）
const sellable = computed(() => {
  if (!selectedMerchant.value?.open) return []
  const cat = selectedMerchant.value.category
  const map = new Map<string, { itemId: string; name: string; price: number; quantity: number }>()
  for (const i of props.inventory) {
    if (i.itemId === 'cash' || i.quantity <= 0 || i.merchantCategory !== cat) continue
    const e = map.get(i.itemId) ?? { itemId: i.itemId, name: i.name, price: i.price, quantity: 0 }
    e.quantity += i.quantity
    map.set(i.itemId, e)
  }
  return [...map.values()]
})

function sellPriceFor(basePrice: number): number {
  const rep = selectedMerchant.value?.reputation ?? 0
  const mult = Math.min(0.45, 0.3 + rep * 0.003)
  return Math.round(basePrice * mult)
}
function qtyFor(itemId: string): number {
  const q = sellQty.value[itemId]
  return q && q > 0 ? q : 1
}
function ownedQty(itemId: string): number {
  return props.inventory.filter((i) => i.itemId === itemId && i.quantity > 0).reduce((s, i) => s + i.quantity, 0)
}

onMounted(() => {
  if (!selectedId.value && props.merchants.length) {
    const firstOpen = props.merchants.find((m) => m.open) ?? props.merchants[0]
    loadCatalog(firstOpen.id)
  }
})
watch(() => props.merchants, (list) => {
  if (!selectedId.value && list.length) {
    const firstOpen = list.find((m) => m.open) ?? list[0]
    loadCatalog(firstOpen.id)
  }
}, { immediate: true })
</script>

<template>
  <section class="view-page merchant-view">
    <header class="page-heading">
      <div><span class="eyebrow">灰区交易频道</span><h1>商人</h1><p>每位商人只买卖自己所属类别的物品，好感度越高价格越优并解锁高级商品。</p></div>
      <div class="channel-state"><span class="status-dot online" />开放交易</div>
    </header>

    <div class="merchant-grid">
      <button
        v-for="m in merchants" :key="m.id" type="button"
        class="merchant-card" :class="{ active: selectedId === m.id, closed: !m.open }"
        @click="loadCatalog(m.id)"
      >
        <div class="merchant-card__head"><strong>{{ m.name }}</strong><span :class="m.open ? 'open' : 'closed'">{{ m.open ? '营业' : '待开放' }}</span></div>
        <p>{{ m.desc }}</p>
        <div class="merchant-card__rep">好感度 <b>{{ m.reputation }}</b></div>
      </button>
    </div>

    <div v-if="selectedMerchant" class="merchant-detail">
      <div class="merchant-detail__title">
        <div><strong>{{ selectedMerchant.name }}</strong><span>{{ selectedMerchant.desc }}</span></div>
        <b>好感度 {{ selectedMerchant.reputation }} <small>影响价格与解锁</small></b>
      </div>

      <div v-if="!selectedMerchant.open" class="merchant-closed">
        <p>该商人暂未开放</p><span>提升商人好感度后可解锁交易</span>
      </div>

      <template v-else>
        <div class="merchant-summary">
          <span>现金 ￥{{ cash.toLocaleString() }}</span><span>{{ catalog.length }} 件在售</span>
        </div>

        <section class="catalog-block">
          <div class="panel-heading"><div><span>SHOP</span><h2>出售给玩家</h2></div><el-button :icon="Refresh" :loading="catalogLoading" circle title="刷新目录" @click="loadCatalog(selectedId)" /></div>
          <div v-if="catalogError" class="text-danger">{{ catalogError }}</div>
          <div v-else class="catalog-list surface-panel">
            <div v-for="item in catalog" :key="item.id" class="catalog-row">
              <div><small>{{ item.kind }}</small><strong>{{ item.name }}</strong><p>{{ item.detail }} · {{ item.weight }}kg/{{ item.slots }}格 · 已有 {{ ownedQty(item.id) }}</p></div>
              <div class="catalog-action">
                <b>￥{{ item.price.toLocaleString() }}</b>
                <small v-if="item.repRequirement > 0">需好感度 {{ item.repRequirement }}</small>
                <el-button type="primary" size="small" :loading="purchasingId === item.id" :disabled="purchasingId !== null || cash < item.price" @click="emit('purchase', selectedId, item.id)">购买</el-button>
              </div>
            </div>
            <el-empty v-if="catalog.length === 0" description="暂无在售商品" />
          </div>
        </section>

        <section class="catalog-block">
          <div class="panel-heading"><div><span>BUYBACK</span><h2>收购玩家</h2></div></div>
          <div class="catalog-list surface-panel">
            <div v-for="item in sellable" :key="item.itemId" class="catalog-row">
              <div><small>可出售</small><strong>{{ item.name }}</strong><p>持有 {{ item.quantity }} · 单价 ￥{{ sellPriceFor(item.price) }}</p></div>
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
