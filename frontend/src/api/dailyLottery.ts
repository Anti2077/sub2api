import { apiClient } from './client'

export interface DailyLotteryPrize {
  id: string
  name: string
  reward_amount: number
  weight: number
  enabled: boolean
}

export interface DailyLotteryPrizeView extends DailyLotteryPrize {
  probability: number
}

export interface DailyLotteryEntry {
  id: number
  user_id: number
  checkin_date: string
  checked_in_at: string
  drawn_at?: string
  prize_id?: string
  prize_name?: string
  reward_amount: number
  created_at: string
  updated_at: string
}

export interface DailyLotteryStatus {
  enabled: boolean
  checked_in: boolean
  can_draw: boolean
  already_drawn: boolean
  today: string
  timezone: string
  next_reset_at: string
  prizes: DailyLotteryPrizeView[]
  entry?: DailyLotteryEntry
}

export interface DailyLotteryDrawResult {
  entry: DailyLotteryEntry
  old_balance: number
  new_balance: number
}

export async function getStatus(): Promise<DailyLotteryStatus> {
  const { data } = await apiClient.get<DailyLotteryStatus>('/daily-lottery/status')
  return data
}

export async function checkIn(): Promise<DailyLotteryStatus> {
  const { data } = await apiClient.post<DailyLotteryStatus>('/daily-lottery/check-in')
  return data
}

export async function draw(): Promise<DailyLotteryDrawResult> {
  const { data } = await apiClient.post<DailyLotteryDrawResult>('/daily-lottery/draw')
  return data
}

export async function getHistory(limit = 30): Promise<DailyLotteryEntry[]> {
  const { data } = await apiClient.get<DailyLotteryEntry[]>('/daily-lottery/history', {
    params: { limit }
  })
  return data
}

export const dailyLotteryAPI = { getStatus, checkIn, draw, getHistory }

export default dailyLotteryAPI
