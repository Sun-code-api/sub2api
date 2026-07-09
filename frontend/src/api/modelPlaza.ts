/**
 * 模型广场 API：以模型为中心聚合定价与分组倍率，按用户可访问分组过滤。
 */

import { apiClient } from './client'
import type { UserAvailableGroup, UserSupportedModelPricing } from './channels'

export interface ModelPlazaEntry {
  name: string
  platform: string
  pricing: UserSupportedModelPricing | null
  groups: UserAvailableGroup[]
}

/** 模型广场（按用户可访问分组过滤，含专属分组）。 */
export async function getModelPlaza(): Promise<ModelPlazaEntry[]> {
  const { data } = await apiClient.get<ModelPlazaEntry[]>('/model-plaza')
  return data || []
}

export const modelPlazaAPI = { getModelPlaza }

export default modelPlazaAPI
