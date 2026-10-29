import { apiClient } from './client'
export interface CardTemplate { id: number; version: number; name: string; kind: 'model_day'|'model_tokens'|'group_rate'; model: string; models?: string[]; group_ids: number[]; rate: number; tokens: number; hours: number; claim_days: number; published: boolean }
export interface RewardCard { id: number; user_id: number; template: CardTemplate; status: string; source: string; remaining_tokens: number; claim_expires_at: string; activated_at?: string; expires_at?: string }
export interface RewardUsage { id: number; request_id: string; model: string; original_cost: number; discount: number; paid_cost: number; tokens: number; covered_tokens: number; created_at: string }
export interface Crop { id: string; name: string; hours: number; yield: number; enabled: boolean; protection_minutes: number; steal_percent: number; max_stolen_percent: number }
export interface RewardConfig { version: number; enabled: boolean; activation_enabled: boolean; farm_enabled: boolean; planting_enabled: boolean; exchange_enabled: boolean; stealing_enabled: boolean; maintenance: boolean; plots: number; protection_minutes: number; steal_percent: number; max_stolen_percent: number; daily_steals: number; daily_per_owner: number; starts_at: string|null; ends_at: string|null; allowed_user_ids: number[]; crops: Crop[] }
export interface Plot { id: number; slot: number; crop: Crop|null; planted_at: string|null; ready_at: string|null; stolen: number; harvested_at: string|null; cycle: string }
export interface Farm { user_id: number; public: boolean; crystals: number; plots: Plot[]; config: RewardConfig; server_time: string }
export interface Shop { id: number; version: number; template_id: number; template: CardTemplate; price: number; stock: number; daily_limit: number; total_limit: number; published: boolean; starts_at: string|null; ends_at: string|null }
export interface Neighbor { user_id: number; nickname: string }
export interface FarmAction { request_key: string; kind: string; slot?: number; crop_id?: string; owner_id?: number; public?: boolean; shop_id?: number }
export interface Ledger { id: number; user_id: number; delta: number; kind: string; reference: string; created_at: string }
export interface Audit { id: number; user_id: number; actor_id: number; card_id: number; kind: string; detail: unknown; created_at: string }
export interface RewardCode {id:number;code:string;status:string;expires_at:string|null;used_by:number|null;used_at:string|null;created_at:string;template:CardTemplate}
const get = async <T>(path: string, admin = false) => (await apiClient.get<T>(`${admin?'/admin':''}/rewards/${path}`)).data
const put = async <T>(path: string, value: T) => (await apiClient.put<T>(`/admin/rewards/${path}`, value)).data
export const rewardsAPI = {
 cards: (admin=false) => get<RewardCard[]>('cards',admin),
 activate: (id:number) => apiClient.post(`/rewards/cards/${id}/activate`),
 records: () => get<RewardUsage[]>('records'),
 templates: () => get<CardTemplate[]>('templates',true), saveTemplate: (t:CardTemplate) => put('templates',t),
 config: () => get<RewardConfig>('config',true), saveConfig: (c:RewardConfig) => put('config',c),
 farm: (owner=0) => get<Farm>(`farm?user_id=${owner}`), neighbors: (q='') => get<Neighbor[]>(`neighbors?search=${encodeURIComponent(q)}`),
 action: async (a:FarmAction) => (await apiClient.post('/rewards/actions',a)).data,
 shop: (admin=false) => get<Shop[]>('shop',admin), saveShop: (s:Shop) => put('shop',s),
 shopStatus: (id:number,version:number,published:boolean) => apiClient.put(`/admin/rewards/shop/${id}/status`,{version,published}),
 ledger: (admin=false) => get<Ledger[]>('farm/ledger',admin), audit: () => get<Audit[]>('records?kind=events',true),
 grant: (user:number,template:number,key:string,reason:string) => apiClient.post('/admin/rewards/grant',{user_id:user,template_id:template,request_key:key,reason}),
 revoke: (id:number,reason:string) => apiClient.post(`/admin/rewards/cards/${id}/revoke`,{reason}),
 listCodes: () => get<RewardCode[]>('codes',true),
 codeStatus: (id:number,status:'unused'|'disabled') => apiClient.put(`/admin/rewards/codes/${id}/status`,{status}),
 codes: async (template:number,count:number,days:number,key?:string) => (await apiClient.post<{codes:string[]}>('/admin/rewards/codes',{template_id:template,count,days,request_key:key})).data.codes
}
export function rewardError(e:unknown):string {const v=e as {response?:{data?:{message?:string}};message?:string};return v.response?.data?.message||v.message||'Error'}
