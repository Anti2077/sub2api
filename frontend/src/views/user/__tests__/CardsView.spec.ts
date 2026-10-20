import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CardsView from '../CardsView.vue'

const api = vi.hoisted(() => ({ cards: vi.fn(), records: vi.fn(), activate: vi.fn() }))
const route = reactive({ query: { tab: 'bag' } })
vi.mock('@/api/rewards', () => ({ rewardsAPI: api, rewardError: (e: Error) => e.message }))
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

const cards = ['unused', 'active', 'expired'].map((status, i) => ({
  id: i + 1, user_id: 1, status, source: 'admin', remaining_tokens: 2500,
  claim_expires_at: '2026-11-01T00:00:00Z',
  template: { id: 1, version: 1, name: `${status} Luna card`, kind: 'model_tokens', model: 'luna', group_ids: [], rate: 1, tokens: 2500, hours: 24, claim_days: 30, published: true }
}))
const render = () => mount(CardsView, { global: { stubs: {
  AppLayout: { template: '<slot />' },
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
  RedeemView: { emits: ['redeemed'], template: '<button data-testid="redeem-done" @click="$emit(\'redeemed\')">Redeem</button>' },
  ConfirmDialog: { props: ['show'], emits: ['confirm', 'cancel'], template: '<div v-if="show" role="dialog"><slot /><button data-testid="confirm" @click="$emit(\'confirm\')">Confirm</button></div>' }
} } })

beforeEach(() => {
  vi.clearAllMocks()
  route.query.tab = 'bag'
  api.cards.mockResolvedValue(cards)
  api.records.mockResolvedValue([])
  api.activate.mockResolvedValue({})
})

describe('CardsView', () => {
  it('shows group, rate and duration before activating a group-rate card', async () => {
    api.cards.mockResolvedValue([{ ...cards[0], template: { ...cards[0].template, kind: 'group_rate', model: '', group_ids: [12, 34], rate: 0.25 } }])
    const page = render()
    await flushPromises()
    await page.get('.reward-ticket button').trigger('click')
    const preview = page.get('[role="dialog"]').text()
    expect(preview).toContain('#12, #34')
    expect(preview).toContain('0.25×')
    expect(preview).toContain('24')
    expect(api.activate).not.toHaveBeenCalled()
    page.unmount()
  })
  it('filters ended cards separately while keeping active cards first', async () => {
    const page = render()
    await flushPromises()
    expect(page.findAll('.reward-ticket h2').map(x => x.text())).toEqual(['active Luna card', 'unused Luna card', 'expired Luna card'])
    await page.findAll('.cards-filter button')[3].trigger('click')
    expect(page.findAll('.reward-ticket')).toHaveLength(1)
    expect(page.get('.reward-ticket h2').text()).toBe('expired Luna card')
    page.unmount()
  })

  it('previews the exact model and quota before activation and blocks a duplicate submit', async () => {
    let complete!: () => void
    api.activate.mockImplementation(() => new Promise<void>(resolve => { complete = resolve }))
    const page = render()
    await flushPromises()
    await page.get('.reward-ticket button').trigger('click')
    expect(api.activate).not.toHaveBeenCalled()
    expect(page.get('[role="dialog"]').text()).toContain('luna')
    expect(page.get('[role="dialog"]').text()).toContain('2,500 Token')
    await page.get('[data-testid="confirm"]').trigger('click')
    expect(api.activate).toHaveBeenCalledTimes(1)
    expect(api.activate).toHaveBeenCalledWith(1)
    expect(page.findAll('.reward-ticket button').every(b => b.attributes('disabled') !== undefined)).toBe(true)
    complete()
    await flushPromises()
    expect(api.cards).toHaveBeenCalledTimes(2)
    page.unmount()
  })

  it('follows query navigation and refreshes the bag after redemption', async () => {
    const page = render()
    await flushPromises()
    route.query.tab = 'redeem'
    await flushPromises()
    await page.get('[data-testid="redeem-done"]').trigger('click')
    await flushPromises()
    expect(api.cards).toHaveBeenCalledTimes(2)
    route.query.tab = 'records'
    await flushPromises()
    expect(page.find('.cards-records').exists()).toBe(true)
    page.unmount()
  })
})
