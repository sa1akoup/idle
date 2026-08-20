<!-- 藏身处页：展示基础设施与已拥有护甲的维修状态。 -->
<script setup lang="ts">
import { Box, FirstAidKit, House, Tools } from '@element-plus/icons-vue'
import type { Armor, ArmorInstance, Player, StorageCapacity } from '../types'

defineProps<{
  player: Player
  armors: Armor[]
  armorInstances: ArmorInstance[]
  repairingId: number | null
  storageCapacity: StorageCapacity | null
}>()

const emit = defineEmits<{
  repair: [id: number]
}>()
</script>

<template>
  <section class="view-page hideout-view">
    <header class="page-heading"><div><span class="eyebrow">安全区域</span><h1>藏身处</h1><p>行动间隙的恢复、整备与库存维护在这里进行。</p></div></header>
    <div class="facility-strip">
      <article><span class="item-icon"><el-icon><Box /></el-icon></span><div><small>储物间</small><strong>等级 1</strong><p>{{ storageCapacity?.capacity ?? 0 }} 个仓位，已用 {{ storageCapacity?.used ?? 0 }}</p></div><i class="online">运行中</i></article>
      <article><span class="item-icon"><el-icon><Tools /></el-icon></span><div><small>维修台</small><strong>等级 1</strong><p>支持一次护甲翻修</p></div><i class="online">运行中</i></article>
      <article><span class="item-icon"><el-icon><FirstAidKit /></el-icon></span><div><small>医疗床</small><strong>基础设施</strong><p>{{ player.injury && player.injury !== 'none' ? '伤势恢复中' : '当前空闲' }}</p></div><i>待命</i></article>
    </div>

    <section class="repair-bench surface-panel">
      <div class="panel-heading"><div><span>BENCH</span><h2>护甲维修台</h2></div><el-icon><House /></el-icon></div>
      <div v-for="instance in armorInstances" :key="instance.id" class="repair-row">
        <div><strong>{{ armors.find((item) => item.id === instance.armorId)?.name || instance.armorId }}</strong><small>维修记录 {{ instance.repairCount }} / 1</small></div>
        <div class="repair-durability">
          <span>{{ instance.curDurability }} / {{ instance.maxDurability }}</span>
          <el-progress :percentage="Math.round((instance.curDurability / instance.maxDurability) * 100)" :show-text="false" />
        </div>
        <el-button
          :icon="Tools"
          :loading="repairingId === instance.id"
          :disabled="instance.curDurability > 0 || instance.repairCount >= 1"
          @click="emit('repair', instance.id)"
        >{{ instance.repairCount >= 1 ? '已翻修' : instance.curDurability > 0 ? '无需维修' : '开始翻修' }}</el-button>
      </div>
    </section>
  </section>
</template>
