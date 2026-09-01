import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TokenLeaderboardView from '../TokenLeaderboardView.vue'

const { getPublicTokenLeaderboard } = vi.hoisted(() => ({
  getPublicTokenLeaderboard: vi.fn()
}))

vi.mock('@/api/usage', () => ({
  usageAPI: { getPublicTokenLeaderboard }
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
    masked_email: 'a***e@example.com',
    requests: 12,
    input_tokens: 100,
    output_tokens: 20,
    cache_tokens: 30,
    total_tokens: 150,
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
  beforeEach(() => {
    getPublicTokenLeaderboard.mockReset()
    getPublicTokenLeaderboard.mockResolvedValue(response)
  })

  it('loads the daily ranking and renders only masked identity data', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getPublicTokenLeaderboard).toHaveBeenCalledWith('day', expect.any(String))
    expect(wrapper.text()).toContain('a***e@example.com')
    expect(wrapper.text()).toContain('2026-09-01 to 2026-09-01')
    expect(wrapper.text()).not.toContain('alice@example.com')
  })

  it('loads a new range when the weekly tab is selected', async () => {
    const wrapper = mountView()
    await flushPromises()

    const weekly = wrapper.findAll('[role="tab"]').find((button) => button.text() === 'Weekly')
    expect(weekly).toBeTruthy()
    await weekly!.trigger('click')
    await flushPromises()

    expect(getPublicTokenLeaderboard).toHaveBeenLastCalledWith('week', expect.any(String))
    expect(weekly!.attributes('aria-selected')).toBe('true')
  })
})
