/**
 * 模型广场 API：以模型为中心聚合定价与分组倍率。
 * 匿名版供落地页与未登录用户浏览；登录版按用户可访问分组过滤。
 */

import { apiClient } from './client'
import type { UserAvailableGroup, UserSupportedModelPricing } from './channels'

export interface ModelPlazaEntry {
  name: string
  platform: string
  pricing: UserSupportedModelPricing | null
  groups: UserAvailableGroup[]
}

/** 匿名版模型广场（仅公开分组）。 */
export async function getPublicModelPlaza(): Promise<ModelPlazaEntry[]> {
  const { data } = await apiClient.get<ModelPlazaEntry[]>('/public/model-plaza')
  return data || []
}

/** 登录版模型广场（按用户可访问分组过滤，含专属分组）。 */
export async function getModelPlaza(): Promise<ModelPlazaEntry[]> {
  const { data } = await apiClient.get<ModelPlazaEntry[]>('/model-plaza')
  return data || []
}

export const modelPlazaAPI = { getPublicModelPlaza, getModelPlaza }

export default modelPlazaAPI
