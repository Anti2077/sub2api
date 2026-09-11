import { apiClient } from './client'
import type { DailyLotteryPrize, DailyLotteryPrizeView } from './dailyLottery'
export interface IncentiveConfig {
  kind: 'global_rate' | 'lottery'; version: number; enabled: boolean; timezone: string
  period: 'weekly'; reset_chances: boolean; group_ids: number[]; excluded_user_ids: number[]
  excluded_models: string[]; exclude_admins: boolean; spend_threshold: number; rate_decrease: number
  minimum_rate: number; max_chances: number; prizes: DailyLotteryPrize[]
}
export interface IncentiveStatus {
  kind: IncentiveConfig['kind']; enabled: boolean; eligible: boolean; period_id: number
  starts_at: string; ends_at: string; timezone: string; spend: number; personal_spend: number
  next_threshold: number; threshold: number; earned: number; used: number; available: number
  groups: { group_id: number; name: string; base_rate: number; current_rate: number }[]
  prizes: DailyLotteryPrizeView[]
}
export interface Reward { id: number; user_id: number; period_id: number; prize: DailyLotteryPrize; reward_amount: number; created_at: string }
export interface IncentiveHistory { rewards: Reward[]; chances: { id: number; user_id: number; chances: number; created_at: string }[]; periods?: { id: number; kind: string; starts_at: string; ends_at: string; spend: number; closed_at: string | null }[] }
export const incentivesAPI = {
  status: async (admin = false) => (await apiClient.get<IncentiveStatus[]>(`${admin ? '/admin' : ''}/incentives/status`)).data,
  history: async (admin = false) => (await apiClient.get<IncentiveHistory>(`${admin ? '/admin' : ''}/incentives/history`)).data,
  configs: async () => (await apiClient.get<IncentiveConfig[]>('/admin/incentives/config')).data,
  save: async (config: IncentiveConfig) => (await apiClient.put<IncentiveConfig>('/admin/incentives/config', config)).data,
  reset: async (id: number) => apiClient.post(`/admin/incentives/periods/${id}/reset`),
  draw: async (requestKey: string) => (await apiClient.post<{ prize: DailyLotteryPrize; reward_amount: number }>('/incentives/draw', { request_key: requestKey })).data
}
