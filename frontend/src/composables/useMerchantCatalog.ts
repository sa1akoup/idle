import { computed, ref, watch, type Ref } from 'vue'
import api, { getApiError } from '../api'
import type { InventoryItem, ItemInstance, Merchant, MerchantCatalogItem, MerchantCatalogResponse } from '../types'

export function formatMerchantRefresh(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

export function useMerchantCatalog(
  merchants: Ref<Merchant[]>,
  inventory: Ref<InventoryItem[]>,
  itemInstances: Ref<ItemInstance[]>,
  purchasingId: Ref<string | null>,
) {
  const selectedId = ref('')
  const catalog = ref<MerchantCatalogItem[]>([])
  const catalogLoading = ref(false)
  const catalogError = ref('')
  const acceptsAny = ref(false)
  const playerSellRate = ref(0.3)
  const nextRefreshAt = ref('')
  const sellQty = ref<Record<string, number>>({})
  const buyQty = ref<Record<string, number>>({})
  let catalogRequestToken = 0

  const selectedMerchant = computed(() => merchants.value.find((item) => item.id === selectedId.value))
  const catalogById = computed(() => new Map(catalog.value.map((item) => [item.id, item])))
  const buyableCatalog = computed(() => catalog.value.filter((item) => item.buyable && (item.stock === undefined || item.stock > 0)))

  async function loadCatalog(id: string) {
    const token = ++catalogRequestToken
    selectedId.value = id
    catalog.value = []
    catalogError.value = ''
    nextRefreshAt.value = ''
    acceptsAny.value = false
    const merchant = merchants.value.find((item) => item.id === id)
    if (!merchant?.open) {
      catalogLoading.value = false
      return
    }
    catalogLoading.value = true
    try {
      const { data } = await api.get<MerchantCatalogResponse>(`/merchants/${id}/catalog`)
      if (token !== catalogRequestToken || selectedId.value !== id) return
      catalog.value = data.items ?? []
      acceptsAny.value = data.acceptsAny
      playerSellRate.value = data.playerSellRate
      nextRefreshAt.value = data.nextRefreshAt ?? ''
      for (const item of catalog.value) {
        if (item.kind === 'ammo' && !buyQty.value[item.id]) buyQty.value[item.id] = Math.min(30, item.stock ?? 30)
      }
    } catch (error) {
      if (token !== catalogRequestToken || selectedId.value !== id) return
      catalogError.value = getApiError(error, '商品目录加载失败')
    } finally {
      if (token === catalogRequestToken && selectedId.value === id) catalogLoading.value = false
    }
  }

  const sellable = computed(() => {
    if (!selectedMerchant.value?.open) return []
    const category = selectedMerchant.value.category
    const map = new Map<string, { itemId: string; name: string; sellPrice: number; quantity: number }>()
    const sellPriceOf = (base: number, catalogItem?: MerchantCatalogItem) => catalogItem?.sellPrice ?? Math.round(base * playerSellRate.value)
    for (const item of inventory.value) {
      if (item.itemId === 'cash' || item.quantity <= 0) continue
      if (!acceptsAny.value && item.merchantCategory !== category) continue
      const catalogItem = catalogById.value.get(item.itemId)
      if (!acceptsAny.value && !catalogItem) continue
      const row = map.get(item.itemId) ?? {
        itemId: item.itemId, name: catalogItem?.name || item.name, sellPrice: sellPriceOf(item.price, catalogItem), quantity: 0,
      }
      row.quantity += item.quantity
      map.set(item.itemId, row)
    }
    for (const instance of itemInstances.value) {
      if (instance.locationType !== 'inventory' || instance.status !== 'normal' || instance.currentDurability <= 0) continue
      if (!acceptsAny.value && instance.merchantCategory !== category) continue
      const catalogItem = catalogById.value.get(instance.itemId)
      if (!acceptsAny.value && !catalogItem) continue
      const row = map.get(instance.itemId) ?? {
        itemId: instance.itemId, name: catalogItem?.name || instance.name || instance.itemId,
        sellPrice: sellPriceOf(instance.price ?? 0, catalogItem), quantity: 0,
      }
      row.quantity += 1
      map.set(instance.itemId, row)
    }
    return [...map.values()]
  })

  function qtyFor(itemId: string): number {
    const quantity = sellQty.value[itemId]
    return quantity && quantity > 0 ? quantity : 1
  }
  function ownedQty(itemId: string): number {
    return inventory.value.filter((item) => item.itemId === itemId && item.quantity > 0).reduce((sum, item) => sum + item.quantity, 0)
      + itemInstances.value.filter((item) => item.itemId === itemId && item.locationType === 'inventory' && item.status === 'normal' && item.currentDurability > 0).length
  }
  function buyQuantity(item: MerchantCatalogItem): number {
    const quantity = buyQty.value[item.id]
    if (quantity && quantity > 0) return quantity
    if (item.kind === 'ammo') return Math.min(30, item.stock ?? 30)
    return 1
  }

  watch(merchants, (list) => {
    if (!selectedId.value && list.length) {
      const firstOpen = list.find((item) => item.open) ?? list[0]
      void loadCatalog(firstOpen.id)
    }
  }, { immediate: true })
  watch(purchasingId, (current, previous) => {
    if (previous && !current && selectedId.value) void loadCatalog(selectedId.value)
  })

  return {
    selectedId, catalogLoading, catalogError, acceptsAny, nextRefreshAt, sellQty, buyQty,
    selectedMerchant, buyableCatalog, sellable, loadCatalog, qtyFor, ownedQty, buyQuantity,
  }
}
