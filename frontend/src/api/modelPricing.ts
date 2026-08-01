import { apiClient } from './client'
import type { ModelPlazaGroup, ModelPlazaResponse } from './modelPlaza'

export interface ModelPricingGroupConfig {
  group_id: number
  models: string[]
}

export interface ModelPricingConfig {
  enabled: boolean
  description: string
  groups: ModelPricingGroupConfig[]
}

export interface ModelPricingAdminResponse {
  config: ModelPricingConfig
  groups: ModelPlazaGroup[]
}

export async function getModelPricing(): Promise<ModelPlazaResponse> {
  const { data } = await apiClient.get<ModelPlazaResponse>('/model-pricing')
  return data
}

export async function getModelPricingConfig(): Promise<ModelPricingAdminResponse> {
  const { data } = await apiClient.get<ModelPricingAdminResponse>('/admin/model-pricing')
  return data
}

export async function updateModelPricingConfig(config: ModelPricingConfig): Promise<ModelPricingConfig> {
  const { data } = await apiClient.put<ModelPricingConfig>('/admin/model-pricing', config)
  return data
}

export const modelPricingAPI = {
  get: getModelPricing,
  getConfig: getModelPricingConfig,
  updateConfig: updateModelPricingConfig
}
