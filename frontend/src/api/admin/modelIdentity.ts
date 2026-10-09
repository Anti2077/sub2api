import { apiClient } from '../client'

export interface IdentityModel { id: string; name: string; family: string }
export interface IdentityConfig { account_id: number; user_id: number; group_id: number; api_key_id: number; key_name: string }
export interface IdentityPlan { id: number; account_id: number; request_model: string; expected_model: string; interval_minutes: number; enabled: boolean; last_run_at?: string; next_run_at?: string }
export interface IdentityProbe { probe_id?: string; status?: number; target_account_id?: number; request_model?: string; prompt?: string; error?: string; evidence?: {upstream_model?: string; request_id?: string}; response?: {usage?: Record<string,unknown>} }
export interface IdentityRun { id: number; plan_id: number; status: string; expected_model: string; request_model: string; report?: Record<string, unknown>; probes?: IdentityProbe[]; probe_count?: number; created_at: string }
export async function models() { const { data } = await apiClient.get<{engine_commit:string;models:IdentityModel[]}>('/admin/model-identity/models'); return data }
export async function config(accountId:number) { const { data } = await apiClient.get<IdentityConfig>(`/admin/model-identity/accounts/${accountId}`); return data }
export async function saveConfig(accountId:number, payload:{user_id:number;group_id:number}) { const { data } = await apiClient.put<IdentityConfig>(`/admin/model-identity/accounts/${accountId}`,payload); return data }
export async function plans(accountId:number) { const { data } = await apiClient.get<IdentityPlan[]>(`/admin/model-identity/accounts/${accountId}/plans`); return data ?? [] }
export async function savePlan(payload:Partial<IdentityPlan>) { const { data } = await apiClient.post<IdentityPlan>('/admin/model-identity/plans',payload); return data }
export async function updatePlan(id:number,payload:Partial<IdentityPlan>) { const { data } = await apiClient.put<IdentityPlan>(`/admin/model-identity/plans/${id}`,payload); return data }
export async function deletePlan(id:number) { await apiClient.delete(`/admin/model-identity/plans/${id}`) }
export async function run(id:number) { const { data } = await apiClient.post<IdentityRun>(`/admin/model-identity/plans/${id}/run`); return data }
export async function history(id:number) { const { data } = await apiClient.get<IdentityRun[]>(`/admin/model-identity/plans/${id}/history`); return data ?? [] }
export async function cancel(id:number) { await apiClient.post(`/admin/model-identity/runs/${id}/cancel`) }
export async function runStatus(id:number) { const { data } = await apiClient.get<IdentityRun>(`/admin/model-identity/runs/${id}`); return data }
