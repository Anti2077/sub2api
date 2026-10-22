import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.hoisted(() => vi.fn())
vi.mock('@/api/client', () => ({ apiClient: { get } }))

import { getCurrentUser } from '@/api/auth'

describe('current user avatar version negotiation', () => {
  beforeEach(() => get.mockReset())

  it('requests a full response when no avatar version is available', async () => {
    get.mockResolvedValue({ data: { id: 1, avatar_url: 'image', avatar_version: 'v1-a' } })
    await getCurrentUser()
    expect(get).toHaveBeenCalledWith('/auth/me', { params: undefined })
  })

  it('sends only the saved version, never the data URI or an HTTP cache validator', async () => {
    const response = { data: { id: 1, balance: 42, avatar_version: 'v1-a' } }
    get.mockResolvedValue(response)
    await expect(getCurrentUser('v1-a')).resolves.toBe(response)
    expect(get).toHaveBeenCalledWith('/auth/me', { params: { avatar_version: 'v1-a' } })
  })
})
