// 前端跨领域共享类型：集中定义行动风格与战利品摘要，避免领域类型互相反向依赖。
export type ActionStyle = 'balanced' | 'stealth' | 'aggressive' | 'greedy'

export interface LootSummary {
  id: string
  itemId: string
  name: string
  category: string
  quantity: number
  containerId: string
  source: string
}
