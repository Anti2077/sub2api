import { apiClient } from '../client'

export interface AccountFolder {
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
  async save(folder: { id?: number; name: string; parent_id: number | null }) {
    return (await apiClient.post<AccountFolder>('/admin/accounts/folders', folder)).data
  },
  async delete(id: number) {
    await apiClient.delete(`/admin/accounts/folders/${id}`)
  },
  async move(account_ids: number[], folder_id: number | null) {
    return (await apiClient.post<{ moved: number }>('/admin/accounts/folders/move', { account_ids, folder_id })).data
  }
}
