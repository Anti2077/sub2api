import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import IncentivesView from '../IncentivesView.vue'
const api = vi.hoisted(() => ({ status: vi.fn(), history: vi.fn(), draw: vi.fn() }))
vi.mock('@/api/incentives', () => ({ incentivesAPI: api }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))
const status = (enabled = true, available = 2) => [{kind:'lottery',enabled,eligible:true,period_id:1,starts_at:'2026-09-07T00:00:00Z',ends_at:'2026-09-14T00:00:00Z',timezone:'UTC',spend:100,personal_spend:100,next_threshold:50,threshold:50,earned:2,used:2-available,available,groups:[],prizes:[]}]
const render = () => mount(IncentivesView, { global: { stubs: { AppLayout: { template:'<slot />' } } } })
beforeEach(() => { vi.clearAllMocks(); api.status.mockResolvedValue(status()); api.history.mockResolvedValue({rewards:[],chances:[]}); api.draw.mockResolvedValue({prize:{name:'Prize'},reward_amount:1}) })
describe('IncentivesView', () => {
 it('allows several earned chances on the same day', async () => { const page=render();await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();expect(api.draw).toHaveBeenCalledTimes(2);expect(api.draw.mock.calls[0][0]).not.toBe(api.draw.mock.calls[1][0]) })
 it('keeps a stable request key after an uncertain draw failure', async () => {api.draw.mockRejectedValueOnce(new Error('network'));const page=render();await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();expect(api.draw.mock.calls[0][0]).toBe(api.draw.mock.calls[1][0])})
 it('does not expose a draw button for a disabled activity', async () => {api.status.mockResolvedValue(status(false));const page=render();await flushPromises();expect(page.find('[data-testid="incentive-draw"]').exists()).toBe(false)})
 it('disables drawing when all chances are used', async () => {api.status.mockResolvedValue(status(true,0));const page=render();await flushPromises();expect(page.get('[data-testid="incentive-draw"]').attributes('disabled')).toBeDefined()})
})
