import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import IncentivesView from '../IncentivesView.vue'
const api = vi.hoisted(() => ({ status: vi.fn(), history: vi.fn(), checkIn: vi.fn(), draw: vi.fn() }))
vi.mock('@/api/incentives', () => ({ incentivesAPI: api }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))
const status = (enabled = true, available = 2, checkInEnabled = false, checkedInToday = false) => [{kind:'lottery',enabled,eligible:true,check_in_enabled:checkInEnabled,checked_in_today:checkedInToday,check_in_chance_awarded:checkedInToday,period_id:1,starts_at:'2026-09-07T00:00:00Z',ends_at:'2026-09-14T00:00:00Z',timezone:'UTC',spend:100,personal_spend:100,next_threshold:50,threshold:50,earned:2,used:2-available,available,groups:[],prizes:[]}]
const render = () => mount(IncentivesView, { global: { stubs: { AppLayout: { template:'<slot />' } } } })
beforeEach(() => { vi.clearAllMocks(); api.status.mockResolvedValue(status()); api.history.mockResolvedValue({rewards:[],chances:[]}); api.checkIn.mockResolvedValue({}); api.draw.mockResolvedValue({prize:{name:'Prize'},reward_amount:1}) })
describe('IncentivesView', () => {
 it('shows the daily check-in action and refreshes the chance count', async () => {api.status.mockResolvedValueOnce(status(true,0,true,false)).mockResolvedValueOnce(status(true,1,true,true));const page=render();await flushPromises();expect(page.get('[data-testid="incentive-check-in"]').attributes('disabled')).toBeUndefined();await page.get('[data-testid="incentive-check-in"]').trigger('click');await flushPromises();expect(api.checkIn).toHaveBeenCalledTimes(1);expect(page.text()).toContain('incentives.checkedInToday')})
 it('allows several earned chances on the same day', async () => { const page=render();await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();expect(api.draw).toHaveBeenCalledTimes(2);expect(api.draw.mock.calls[0][0]).not.toBe(api.draw.mock.calls[1][0]) })
 it('keeps a stable request key after an uncertain draw failure', async () => {api.draw.mockRejectedValueOnce(new Error('network'));const page=render();await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();await page.get('[data-testid="incentive-draw"]').trigger('click');await flushPromises();expect(api.draw.mock.calls[0][0]).toBe(api.draw.mock.calls[1][0])})
 it('disables the draw button when the activity is disabled', async () => {api.status.mockResolvedValue(status(false));const page=render();await flushPromises();expect(page.get('[data-testid="incentive-draw"]').attributes('disabled')).toBeDefined()})
 it('disables drawing when all chances are used', async () => {api.status.mockResolvedValue(status(true,0));const page=render();await flushPromises();expect(page.get('[data-testid="incentive-draw"]').attributes('disabled')).toBeDefined()})
})
