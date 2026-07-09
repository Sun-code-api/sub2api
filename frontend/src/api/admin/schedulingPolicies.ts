/**
 * Admin Scheduling Policy API endpoints
 * Monitor-driven channel/account scheduling automation (pause / deprioritize / recover)
 */

import { apiClient } from '../client'

export type SchedulingActionType = 'pause' | 'deprioritize'
export type SchedulingActionKind = 'pause' | 'deprioritize' | 'recover'

export interface SchedulingPolicy {
  id: number
  name: string
  enabled: boolean
  monitor_id: number
  account_ids: number[]
  trigger_consecutive_failures: number
  trigger_latency_ms: number
  action_type: SchedulingActionType
  pause_minutes: number
  priority_delta: number
  recover_consecutive_successes: number
  cooldown_minutes: number
  created_at: string
  updated_at: string
}

export interface SchedulingPolicyParams {
  name: string
  enabled?: boolean
  monitor_id: number
  account_ids: number[]
  trigger_consecutive_failures: number
  trigger_latency_ms?: number
  action_type: SchedulingActionType
  pause_minutes?: number
  priority_delta?: number
  recover_consecutive_successes?: number
  cooldown_minutes?: number
}

export interface SchedulingPolicyListParams {
  page?: number
  page_size?: number
  enabled?: boolean
  search?: string
}

export interface SchedulingPolicyListResponse {
  items: SchedulingPolicy[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface SchedulingAction {
  id: number
  policy_id: number
  account_id: number
  monitor_id: number
  action: SchedulingActionKind
  reason: string
  original_priority: number
  restored: boolean
  created_at: string
}

export interface SchedulingActionListParams {
  page?: number
  page_size?: number
  policy_id?: number
  account_id?: number
}

export interface SchedulingActionListResponse {
  items: SchedulingAction[]
  total: number
  page: number
  page_size: number
  pages: number
}

export async function list(
  params: SchedulingPolicyListParams = {}
): Promise<SchedulingPolicyListResponse> {
  const { data } = await apiClient.get<SchedulingPolicyListResponse>('/admin/scheduling-policies', {
    params,
  })
  return data
}

export async function get(id: number): Promise<SchedulingPolicy> {
  const { data } = await apiClient.get<SchedulingPolicy>(`/admin/scheduling-policies/${id}`)
  return data
}

export async function create(params: SchedulingPolicyParams): Promise<SchedulingPolicy> {
  const { data } = await apiClient.post<SchedulingPolicy>('/admin/scheduling-policies', params)
  return data
}

export async function update(id: number, params: SchedulingPolicyParams): Promise<SchedulingPolicy> {
  const { data } = await apiClient.put<SchedulingPolicy>(`/admin/scheduling-policies/${id}`, params)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/scheduling-policies/${id}`)
}

export async function listActions(
  params: SchedulingActionListParams = {}
): Promise<SchedulingActionListResponse> {
  const { data } = await apiClient.get<SchedulingActionListResponse>(
    '/admin/scheduling-policies/actions',
    { params }
  )
  return data
}

export const schedulingPoliciesAPI = {
  list,
  get,
  create,
  update,
  remove,
  listActions,
}

export default schedulingPoliciesAPI
