<!-- 仓库页：展示所有物品、装备占用和现金余额，装备不会从仓库中隐形消失。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { Box, Coin, Goods, Suitcase } from '@element-plus/icons-vue'
import type { InventoryItem, ItemInstance, PlayerLoadout, StorageCapacity } from '../types'

const props = defineProps<{ inventory: InventoryItem[]; itemInstances: ItemInstance[]; loadout: PlayerLoadout | null; storageCapacity: StorageCapacity | null }>()
const nonCashItems = computed(() => props.inventory.filter((item) => item.itemId !== 'cash'))
const inventoryInstances = computed(() => props.itemInstances.filter((item) => item.locationType === 'inventory'))
const totalUnits = computed(() => nonCashItems.value.reduce((sum, item) => sum + item.quantity, 0) + inventoryInstances.value.length)
const equippedIds = computed(() => {
  if (!props.loadout) return new Set<string>()
  return new Set([
    props.loadout.weaponId, props.loadout.armorId, props.loadout.chestRigId, props.loadout.backpackId,
    props.loadout.helmetId, props.loadout.headsetId, ...props.loadout.consumables,
  ])
})
const kindNames: Record<InventoryItem['kind'], string> = {
  currency: '资金', material: '材料', loot: '战利品', weapon: '武器', armor: '护甲', ammo: '弹药', consumable: '补给',
  chestrig: '胸挂', backpack: '背包', helmet: '头盔', headset: '耳机',
}
const lootCategoryNames: Record<string, string> = {
  tool: '工具', material: '建材', electronics: '电子', info: '情报', medical: '医疗', food: '食品', valuable: '贵重物', fuel: '燃料', weaponpart: '武器零件',
}

function ammoSlots(quantity: number): number {
  return Math.ceil(quantity / 999)
}
</script>

<template>
  <section class="view-page inventory-view">
    <header class="page-heading"><div><span class="eyebrow">本地库存</span><h1>仓库</h1><p>装备、补给和行动带回的物资统一占用仓位。</p></div></header>
    <div class="summary-strip">
      <div><el-icon><Coin /></el-icon><span>现金余额</span><strong>￥{{ (inventory.find((item) => item.itemId === 'cash')?.quantity ?? 0).toLocaleString() }}</strong></div>
      <div><el-icon><Goods /></el-icon><span>物品数量</span><strong>{{ totalUnits }}</strong></div>
      <div><el-icon><Box /></el-icon><span>占用仓位</span><strong>{{ storageCapacity?.used ?? 0 }} / {{ storageCapacity?.capacity ?? 0 }}</strong></div>
    </div>

    <section class="inventory-table surface-panel">
      <div class="panel-heading"><div><span>STASH</span><h2>物资清单</h2></div><small>{{ inventory.length + inventoryInstances.length }} 类物资</small></div>
      <div class="data-list-header inventory-header"><span>物资</span><span>类型</span><span>数量</span><span>重量/格数</span><span>状态</span><span>估值</span></div>
      <div v-for="item in inventory" :key="item.id" class="data-list-row inventory-row">
        <div class="item-name"><span class="item-icon"><el-icon><Coin v-if="item.itemId === 'cash'" /><Box v-else /></el-icon></span><div><strong>{{ item.name }}</strong><small>{{ item.itemId }}</small></div></div>
        <span>{{ item.kind === 'loot' ? lootCategoryNames[item.category] ?? kindNames[item.kind] : kindNames[item.kind] }}</span><b>× {{ item.quantity }}</b>
        <span class="text-muted">{{ item.kind === 'ammo' ? `${ammoSlots(item.quantity)} 格 · 每格最多 999 发` : `${item.weight}kg / ${item.slots}格` }}</span>
        <span v-if="equippedIds.has(item.itemId)" class="equipped-label"><el-icon><Suitcase /></el-icon>已装备</span><span v-else class="text-muted">仓库存放</span>
        <strong>￥{{ (item.quantity * item.price).toLocaleString() }}</strong>
      </div>
      <div v-for="instance in inventoryInstances" :key="`instance-${instance.id}`" class="data-list-row inventory-row">
        <div class="item-name"><span class="item-icon"><el-icon><Box /></el-icon></span><div><strong>{{ instance.name ?? instance.itemId }}</strong><small>{{ instance.itemId }} · 实例 #{{ instance.id }}</small></div></div>
        <span>{{ instance.category ? lootCategoryNames[instance.category] ?? instance.kind ?? '耐久物品' : instance.kind ?? '耐久物品' }}</span><b>× 1</b>
        <span class="text-muted">{{ instance.weight ?? 0 }}kg / {{ instance.slots ?? 0 }}格</span>
        <span :class="instance.status === 'normal' ? 'text-muted' : 'text-danger'">耐久 {{ Math.round(instance.currentDurability) }} / {{ Math.round(instance.maxDurability) }}</span>
        <strong>￥{{ (instance.price ?? 0).toLocaleString() }}</strong>
      </div>
    </section>
  </section>
</template>
