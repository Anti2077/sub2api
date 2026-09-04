import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UserUsageEquivalence from '../UserUsageEquivalence.vue'
import type { UsageEquivalenceResponse } from '@/api/usage'

const { getUsageEquivalence } = vi.hoisted(() => ({
  getUsageEquivalence: vi.fn()
}))

vi.mock('@/api/usage', () => ({
  usageAPI: { getUsageEquivalence }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en' },
      t: (key: string) => key
    })
  }
})

const response = (
  overrides: Partial<UsageEquivalenceResponse> = {}
): UsageEquivalenceResponse => ({
  period: 'this_month',
  start_time: '2026-09-01T00:00:00+08:00',
  end_time: '2026-09-04T12:00:00+08:00',
  timezone: 'Asia/Shanghai',
  scope: 'all_models',
  currency: 'USD',
  standard_cost: 260,
  actual_cost: 78,
  effective_rate_multiplier: 0.3,
  total_requests: 11,
  total_tokens: 9000,
  plans: [
    { id: 'chatgpt_plus', name: 'ChatGPT Plus', monthly_price: 20, usage_multiple: 1, equivalent_months: 13 },
    { id: 'chatgpt_pro_5x', name: 'ChatGPT Pro 5x', monthly_price: 100, usage_multiple: 5, equivalent_months: 2.6 },
    { id: 'chatgpt_pro_20x', name: 'ChatGPT Pro 20x', monthly_price: 200, usage_multiple: 20, equivalent_months: 1.3 }
  ],
  pricing_basis: 'recorded_standard_cost',
  pricing_as_of: '2026-09-04',
  pricing_source: 'https://learn.chatgpt.com/docs/pricing',
  disclaimer: 'api_price_equivalent_not_quota_measurement',
  ...overrides
})

const deferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

const mountComponent = () => mount(UserUsageEquivalence, {
  global: {
    stubs: {
      Icon: true,
      Skeleton: { template: '<div data-testid="skeleton" />' }
    }
  }
})

describe('UserUsageEquivalence', () => {
  beforeEach(() => {
    getUsageEquivalence.mockReset()
  })

  it('loads this month by default and renders all OpenAI plan equivalents', async () => {
    getUsageEquivalence.mockResolvedValue(response())

    const wrapper = mountComponent()
    await flushPromises()

    expect(getUsageEquivalence).toHaveBeenCalledTimes(1)
    expect(getUsageEquivalence).toHaveBeenCalledWith(
      'this_month',
      expect.any(String),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(wrapper.get('[data-testid="usage-equivalence-plan-chatgpt_plus"]').text()).toContain('13')
    expect(wrapper.get('[data-testid="usage-equivalence-plan-chatgpt_pro_5x"]').text()).toContain('2.6')
    expect(wrapper.get('[data-testid="usage-equivalence-plan-chatgpt_pro_20x"]').text()).toContain('1.3')
  })

  it('reloads an exact preset when the period changes', async () => {
    getUsageEquivalence.mockResolvedValue(response())
    const wrapper = mountComponent()
    await flushPromises()

    await wrapper.get('[data-testid="usage-equivalence-period-last_24h"]').trigger('click')
    await flushPromises()

    expect(getUsageEquivalence).toHaveBeenLastCalledWith(
      'last_24h',
      expect.any(String),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })

  it('shows the zero state without hiding the three plans', async () => {
    getUsageEquivalence.mockResolvedValue(response({
      standard_cost: 0,
      actual_cost: 0,
      effective_rate_multiplier: 0,
      total_requests: 0,
      total_tokens: 0,
      plans: response().plans.map((plan) => ({ ...plan, equivalent_months: 0 }))
    }))

    const wrapper = mountComponent()
    await flushPromises()

    expect(wrapper.get('[data-testid="usage-equivalence-zero"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid^="usage-equivalence-plan-"]')).toHaveLength(3)
  })

  it('shows an error and retries the current period', async () => {
    getUsageEquivalence
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(response())

    const wrapper = mountComponent()
    await flushPromises()
    expect(wrapper.get('[data-testid="usage-equivalence-error"]').exists()).toBe(true)

    await wrapper.get('[data-testid="usage-equivalence-error"] button').trigger('click')
    await flushPromises()

    expect(getUsageEquivalence).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-testid="usage-equivalence-error"]').exists()).toBe(false)
  })

  it('does not let an older request replace a newer period', async () => {
    const first = deferred<UsageEquivalenceResponse>()
    const second = deferred<UsageEquivalenceResponse>()
    getUsageEquivalence
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)

    const wrapper = mountComponent()
    await wrapper.get('[data-testid="usage-equivalence-period-last_7d"]').trigger('click')

    second.resolve(response({
      period: 'last_7d',
      standard_cost: 40,
      plans: response().plans.map((plan) => ({ ...plan, equivalent_months: 2 }))
    }))
    await flushPromises()
    expect(wrapper.get('[data-testid="usage-equivalence-plan-chatgpt_plus"]').text()).toContain('2')

    first.resolve(response({
      standard_cost: 999,
      plans: response().plans.map((plan) => ({ ...plan, equivalent_months: 49.95 }))
    }))
    await flushPromises()
    expect(wrapper.get('[data-testid="usage-equivalence-plan-chatgpt_plus"]').text()).not.toContain('49.95')
  })
})
