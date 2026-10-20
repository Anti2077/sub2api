import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FarmView from '../FarmView.vue'

const api = vi.hoisted(() => ({ farm: vi.fn(), shop: vi.fn(), ledger: vi.fn(), action: vi.fn(), neighbors: vi.fn() }))
vi.mock('@/api/rewards', () => ({ rewardsAPI: api, rewardError: (e: Error) => e.message }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))
const crop = { id: 'sprout', name: 'Mint crystals', hours: 1, yield: 10, enabled: true, protection_minutes: 30, steal_percent: 10, max_stolen_percent: 40 }
const template = { id: 1, name: 'Luna Token card', kind: 'model_tokens', model: 'luna', group_ids: [12], tokens: 2500, hours: 24 }
const item = { id: 1, version: 1, template, price: 20, stock: 10, daily_limit: 3, total_limit: 30 }
const fixture = () => ({
  user_id: 1, crystals: 100, public: true, server_time: '2026-10-19T00:00:00Z',
  config: { crops: [crop] },
  plots: [
    { slot: 1, crop, planted_at: '2026-10-18T23:30:00Z', ready_at: '2026-10-19T00:30:00Z', harvested_at: null, stolen: 2, cycle: 'growing' },
    { slot: 2, crop: null, cycle: 'empty' }
  ]
})
const render = () => mount(FarmView, { global: { stubs: {
  AppLayout: { template: '<slot />' },
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
  ConfirmDialog: { props: ['show'], emits: ['confirm', 'cancel'], template: '<div v-if="show" role="dialog"><slot /><button data-testid="confirm" @click="$emit(\'confirm\')">Confirm</button><button data-testid="cancel" @click="$emit(\'cancel\')">Cancel</button></div>' }
} } })

beforeEach(() => {
  vi.clearAllMocks()
  api.farm.mockResolvedValue(fixture())
  api.shop.mockResolvedValue([item])
  api.ledger.mockResolvedValue([])
  api.neighbors.mockResolvedValue([{ user_id: 2, nickname: 'Neighbor' }])
  api.action.mockResolvedValue({})
})
describe('FarmView', () => {
  it('keeps the scene and soil mounted during an action and refresh, and ignores repeated clicks', async () => {
    let finishAction!: () => void
    let finishRefresh!: (farm: ReturnType<typeof fixture>) => void
    api.action.mockImplementation(() => new Promise<void>(resolve => { finishAction = resolve }))
    api.farm.mockResolvedValueOnce(fixture()).mockImplementationOnce(() => new Promise(resolve => { finishRefresh = resolve }))
    const page = render()
    await flushPromises()
    const scene = page.get('.farm-scene').element
    const soil = page.findAll('.pod-planter').map(p => p.element)
    const empty = page.findAll('.farm-plot')[1]
    await empty.trigger('click')
    await empty.trigger('click')
    expect(api.action).toHaveBeenCalledTimes(1)
    expect(empty.attributes('aria-disabled')).toBe('true')
    expect(empty.attributes('aria-busy')).toBe('true')
    expect(empty.attributes('disabled')).toBeUndefined()
    finishAction()
    await flushPromises()
    expect(page.get('.farm-scene').element).toBe(scene)
    expect(page.find('.farm-loading-skeleton').exists()).toBe(false)
    expect(page.get('.farm-messages [role="status"]').exists()).toBe(true)
    finishRefresh(fixture())
    await flushPromises()
    expect(page.get('.farm-scene').element).toBe(scene)
    page.findAll('.pod-planter').forEach((p, i) => expect(p.element).toBe(soil[i]))
    expect(empty.attributes('aria-disabled')).toBe('false')
    page.unmount()
  })

  it('inspects a growing crop with remaining yield and progress without mutating assets', async () => {
    const page = render()
    await flushPromises()
    await page.get('.farm-plot').trigger('click')
    expect(page.get('.farm-crop-detail').text()).toContain('8 rewards.crystals')
    expect(page.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('50')
    expect(api.action).not.toHaveBeenCalled()
    page.unmount()
  })

  it('previews cost and rights, cancels without deduction, then confirms exactly once and links to the bag', async () => {
    let complete!: () => void
    api.action.mockImplementation(() => new Promise<void>(resolve => { complete = resolve }))
    const page = render()
    await flushPromises()
    await page.findAll('.rw-segments button')[2].trigger('click')
    await flushPromises()
    await page.get('.farm-shop-card button').trigger('click')
    const preview = page.get('[role="dialog"]').text()
    expect(preview).toContain('luna')
    expect(preview).toContain('#12')
    expect(preview).toContain('2,500 Token')
    expect(preview).toContain('−20')
    expect(preview).toContain('80 rewards.crystals')
    expect(api.action).not.toHaveBeenCalled()
    await page.get('[data-testid="cancel"]').trigger('click')
    expect(api.action).not.toHaveBeenCalled()
    await page.get('.farm-shop-card button').trigger('click')
    await page.get('[data-testid="confirm"]').trigger('click')
    expect(api.action).toHaveBeenCalledTimes(1)
    expect(page.get('.farm-shop-card button').attributes('disabled')).toBeDefined()
    complete()
    await flushPromises()
    expect(page.get('.shop-receipt a').attributes('href')).toBe('/cards?tab=bag')
    page.unmount()
  })

  it('reuses an uncertain exchange request key and creates a new key after success', async () => {
    api.action.mockRejectedValueOnce(new Error('network'))
    const page = render()
    await flushPromises()
    await page.findAll('.rw-segments button')[2].trigger('click')
    await flushPromises()
    for (let i = 0; i < 3; i++) {
      await page.get('.farm-shop-card button').trigger('click')
      await page.get('[data-testid="confirm"]').trigger('click')
      await flushPromises()
    }
    expect(api.action.mock.calls[0][0].request_key).toBe(api.action.mock.calls[1][0].request_key)
    expect(api.action.mock.calls[1][0].request_key).not.toBe(api.action.mock.calls[2][0].request_key)
    page.unmount()
  })

  it('hides seed selection when visiting another farm and offers a return path', async () => {
    const page = render()
    await flushPromises()
    await page.findAll('.rw-segments button')[1].trigger('click')
    await flushPromises()
    await page.get('.neighbor-card button').trigger('click')
    await flushPromises()
    expect(api.farm).toHaveBeenLastCalledWith(2)
    expect(page.find('.farm-seeds').exists()).toBe(false)
    expect(page.get('.farm-scene-top button').text()).toBe('rewards.back')
    expect(api.action).not.toHaveBeenCalled()
    page.unmount()
  })
})
