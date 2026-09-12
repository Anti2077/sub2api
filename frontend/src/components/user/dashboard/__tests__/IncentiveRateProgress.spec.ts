import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import IncentiveRateProgress from '../IncentiveRateProgress.vue'

const api = vi.hoisted(() => ({
  status: vi.fn()
}))

vi.mock('@/api/incentives', () => ({ incentivesAPI: api }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const makeStatus = (overrides: Record<string, unknown> = {}) => ({
  kind: 'global_rate' as const,
  enabled: true,
  eligible: true,
  period_id: 1,
  starts_at: '2026-09-07T00:00:00Z',
  ends_at: '2026-09-14T00:00:00Z',
  timezone: 'UTC',
  spend: 55.91,
  personal_spend: 0,
  next_threshold: 44.09,
  threshold: 50,
  earned: 0,
  used: 0,
  available: 0,
  groups: [{ group_id: 1, name: 'GPT', base_rate: 0.45, current_rate: 0.42 }],
  prizes: [],
  ...overrides
})

const mountComponent = () => mount(IncentiveRateProgress, {
  global: {
    stubs: {
      RouterLink: { template: '<a><slot /></a>' },
      Icon: { template: '<span />' }
    }
  }
})

describe('IncentiveRateProgress', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.status.mockResolvedValue([makeStatus()])
  })

  it('keeps the dashboard card visible for excluded users and shows the final rate', async () => {
    api.status.mockResolvedValue([makeStatus({ eligible: false })])

    const wrapper = mountComponent()
    await flushPromises()

    expect(wrapper.find('section').exists()).toBe(true)
    expect(wrapper.text()).toContain('0.4200x')
    expect(wrapper.text()).toContain('incentives.excludedNotice')
  })

  it('hides the card when the activity is disabled', async () => {
    api.status.mockResolvedValue([makeStatus({ enabled: false })])

    const wrapper = mountComponent()
    await flushPromises()

    expect(wrapper.find('section').exists()).toBe(false)
  })
})
