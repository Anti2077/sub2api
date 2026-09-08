import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get }
}))

import { getAvailableModels } from '@/api/admin/accounts'

describe('admin account models API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('falls back to the model id when the upstream omits display_name', async () => {
    get.mockResolvedValueOnce({
      data: [
        { id: 'gpt-6-astra' },
        { id: 'gpt-5.4', display_name: 'GPT-5.4' },
        { id: 'gpt-5.3', display_name: '   ' }
      ]
    })

    await expect(getAvailableModels(42)).resolves.toEqual([
      { id: 'gpt-6-astra', type: 'model', display_name: 'gpt-6-astra', created_at: '' },
      { id: 'gpt-5.4', display_name: 'GPT-5.4', type: 'model', created_at: '' },
      { id: 'gpt-5.3', display_name: 'gpt-5.3', type: 'model', created_at: '' }
    ])
    expect(get).toHaveBeenCalledWith('/admin/accounts/42/models')
  })
})
