import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TokenLeaderboardView from '../TokenLeaderboardView.vue'

const { getPublicTokenLeaderboard, getLeaderboardAvatar } = vi.hoisted(() => ({
  getPublicTokenLeaderboard: vi.fn(),
  getLeaderboardAvatar: vi.fn()
}))

vi.mock('@/api/usage', () => ({
  usageAPI: { getPublicTokenLeaderboard },
  getLeaderboardAvatar
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'leaderboard.day') return 'Daily'
        if (key === 'leaderboard.week') return 'Weekly'
        if (key === 'leaderboard.month') return 'Monthly'
        if (key === 'leaderboard.year') return 'Yearly'
        if (key === 'leaderboard.dateRange') return `${params?.start} to ${params?.end}`
        return key
      }
    })
  }
})

const response = {
  ranking: [{
    rank: 1,
    username: 'alice',
    avatar_url: 'https://cdn.example.com/alice.png',
    is_anonymous: false,
    requests: 12,
    input_tokens: 100,
    output_tokens: 20,
    cache_tokens: 30,
    total_tokens: 150,
    actual_cost: 2.5,
    is_current_user: true
  }],
  period: 'day' as const,
  start_date: '2026-09-01',
  end_date: '2026-09-01'
}

function mountView() {
  return mount(TokenLeaderboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: { template: '<span class="icon" />' }
      }
    }
  })
}

describe('TokenLeaderboardView', () => {
  afterEach(() => vi.unstubAllGlobals())
  beforeEach(() => {
    getLeaderboardAvatar.mockReset()
    vi.stubGlobal('URL', Object.assign(URL, {
      createObjectURL: vi.fn(() => 'blob:leaderboard-image'),
      revokeObjectURL: vi.fn()
    }))
    getPublicTokenLeaderboard.mockReset()
    getPublicTokenLeaderboard.mockResolvedValue(response)
  })

  it('loads the daily ranking and renders usernames without email data', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getPublicTokenLeaderboard).toHaveBeenCalledWith('day', expect.any(String), 'tokens')
    expect(wrapper.text()).toContain('alice')
    expect(wrapper.text()).toContain('2026-09-01 to 2026-09-01')
    expect(wrapper.text()).not.toContain('@example.com')
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-image"]')).toHaveLength(2)
    expect(wrapper.get('[data-testid="leaderboard-avatar-image"]').attributes('src')).toBe('https://cdn.example.com/alice.png')
  })

  it('uses a stable initial fallback when an avatar cannot be loaded', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="leaderboard-avatar-image"]').trigger('error')

    expect(wrapper.findAll('[data-testid="leaderboard-avatar-image"]')).toHaveLength(0)
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-fallback"]')).toHaveLength(2)
    expect(wrapper.get('[data-testid="leaderboard-avatar-fallback"]').text()).toBe('A')
  })

  it('never renders an avatar URL for an anonymous ranking entry', async () => {
    getPublicTokenLeaderboard.mockResolvedValue({
      ...response,
      ranking: [{ ...response.ranking[0], username: '', is_anonymous: true }]
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="leaderboard-avatar-image"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-fallback"]')).toHaveLength(2)
  })

  it('loads a new range when the weekly tab is selected', async () => {
    const wrapper = mountView()
    await flushPromises()

    const weekly = wrapper.findAll('[role="tab"]').find((button) => button.text() === 'Weekly')
    expect(weekly).toBeTruthy()
    await weekly!.trigger('click')
    await flushPromises()

    expect(getPublicTokenLeaderboard).toHaveBeenLastCalledWith('week', expect.any(String), 'tokens')
    expect(weekly!.attributes('aria-selected')).toBe('true')
  })

  it('switches between token and spending rankings', async () => {
    const wrapper = mountView()
    await flushPromises()

    const spending = wrapper.find('button[aria-pressed="false"]')
    await spending.trigger('click')
    await flushPromises()

    expect(getPublicTokenLeaderboard).toHaveBeenLastCalledWith('day', expect.any(String), 'spending')
  })
  it('renders the ranking before avatar downloads complete and shares blobs across layouts', async () => {
    let finish!: (blob: Blob) => void
    getLeaderboardAvatar.mockReturnValue(new Promise<Blob>((resolve) => { finish = resolve }))
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], avatar_url: undefined, avatar_version: 'v1'
    }] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('alice')
    expect(wrapper.text()).toContain('150')
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-fallback"]')).toHaveLength(2)
    expect(getLeaderboardAvatar).toHaveBeenCalledTimes(1)
    finish(new Blob(['GIF89a'], { type: 'image/gif' }))
    await flushPromises()
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-image"]')).toHaveLength(2)
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1)
    await wrapper.findAll('[role="tab"]')[1]!.trigger('click')
    await flushPromises()
    expect(getLeaderboardAvatar).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:leaderboard-image')
  })

  it('keeps ranking visible if image retrieval fails', async () => {
    getLeaderboardAvatar.mockRejectedValue(new Error('image unavailable'))
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], avatar_url: undefined, avatar_version: 'v1'
    }] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('alice')
    expect(wrapper.text()).not.toContain('leaderboard.loadFailed')
    expect(wrapper.findAll('[data-testid="leaderboard-avatar-fallback"]')).toHaveLength(2)
    wrapper.unmount()
  })

  it('bounds concurrent images and cancels downloads when leaving the page', async () => {
    const signals: AbortSignal[] = []
    getLeaderboardAvatar.mockImplementation((_name, _version, signal) => {
      signals.push(signal)
      return new Promise(() => {})
    })
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: Array.from({ length: 8 }, (_, i) => ({
      ...response.ranking[0], username: `user${i}`, rank: i + 1, avatar_url: undefined, avatar_version: `v${i}`
    })) })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('user7')
    expect(getLeaderboardAvatar).toHaveBeenCalledTimes(3)
    wrapper.unmount()
    expect(signals.every((signal) => signal.aborted)).toBe(true)
  })

  it('loads changed versions and revokes old images when an entry becomes anonymous', async () => {
    getLeaderboardAvatar.mockResolvedValue(new Blob(['image'], { type: 'image/png' }))
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], avatar_url: undefined, avatar_version: 'v1'
    }] })
    const wrapper = mountView()
    await flushPromises()
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], avatar_url: undefined, avatar_version: 'v2'
    }] })
    await wrapper.findAll('[role="tab"]')[1]!.trigger('click')
    await flushPromises()
    expect(getLeaderboardAvatar).toHaveBeenLastCalledWith('alice', 'v2', expect.any(AbortSignal))
    expect(URL.revokeObjectURL).toHaveBeenCalledTimes(1)
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], username: '', is_anonymous: true, avatar_version: 'v2'
    }] })
    await wrapper.findAll('[role="tab"]')[2]!.trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="leaderboard-avatar-image"]').exists()).toBe(false)
    expect(URL.revokeObjectURL).toHaveBeenCalledTimes(2)
    expect(getLeaderboardAvatar).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('ignores late image responses after a range change', async () => {
    let finish!: (blob: Blob) => void
    getLeaderboardAvatar.mockReturnValue(new Promise<Blob>((resolve) => { finish = resolve }))
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [{
      ...response.ranking[0], avatar_url: undefined, avatar_version: 'v1'
    }] })
    const wrapper = mountView()
    await flushPromises()
    getPublicTokenLeaderboard.mockResolvedValue({ ...response, ranking: [] })
    await wrapper.findAll('[role="tab"]')[1]!.trigger('click')
    await flushPromises()
    finish(new Blob(['image'], { type: 'image/png' }))
    await flushPromises()
    expect(URL.createObjectURL).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="leaderboard-avatar-image"]').exists()).toBe(false)
    wrapper.unmount()
  })

})
