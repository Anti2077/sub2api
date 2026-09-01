import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { DailyLotteryEntry, DailyLotteryPrize } from '../dailyLottery'

export interface DailyLotteryConfig {
  enabled: boolean
  prizes: DailyLotteryPrize[]
}

export interface DailyLotteryAdminEntry extends DailyLotteryEntry {
  user_email: string
  username: string
}

export async function getConfig(): Promise<DailyLotteryConfig> {
  const { data } = await apiClient.get<DailyLotteryConfig>('/admin/daily-lottery/config')
  return data
}

export async function updateConfig(config: DailyLotteryConfig): Promise<DailyLotteryConfig> {
  const { data } = await apiClient.put<DailyLotteryConfig>('/admin/daily-lottery/config', config)
  return data
}

export async function getHistory(page = 1, pageSize = 20): Promise<PaginatedResponse<DailyLotteryAdminEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<DailyLotteryAdminEntry>>('/admin/daily-lottery/history', {
    params: { page, page_size: pageSize }
  })
  return data
}

const dailyLotteryAdminAPI = { getConfig, updateConfig, getHistory }

export default dailyLotteryAdminAPI
