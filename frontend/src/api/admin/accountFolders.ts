import { apiClient } from '../client'

export interface AccountFolderRules {
  platforms?: string[]
  tags?: string[]
  tag_match?: 'all' | 'any'
  tag_state?: '' | 'tagged' | 'untagged'
  name?: string
  types?: string[]
  statuses?: string[]
  schedulable?: boolean
  expiry?: '' | 'expired' | 'expiring_7d' | 'never'
}
export interface AccountTag { name: string; account_count: number }
export interface AccountFolder {
  rules: AccountFolderRules
  id: number
  name: string
  parent_id: number | null
  account_count: number
  error_count: number
}
export interface AccountFolderOverview {
  folders: AccountFolder[]
  total: number
  unfiled: number
}
export const accountFoldersAPI = {
  async overview() {
    return (await apiClient.get<AccountFolderOverview>('/admin/accounts/folders')).data
  },
  async save(folder: { id?: number; name: string; parent_id: number | null; rules: AccountFolderRules }) {
    return (await apiClient.post<AccountFolder>('/admin/accounts/folders', folder)).data
  },
  async delete(id: number) {
    await apiClient.delete(`/admin/accounts/folders/${id}`)
  },
  async tags() {
    return (await apiClient.get<AccountTag[]>('/admin/accounts/tags')).data
  },
  async updateTags(account_ids: number[], tags: string[], operation: 'add' | 'remove') {
    return (await apiClient.post<{ updated: number }>('/admin/accounts/tags', { account_ids, tags, operation })).data
  }
}
