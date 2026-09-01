import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DailyLotteryView from '../DailyLotteryView.vue'

const mocks = vi.hoisted(() => ({
  getStatus: vi.fn(),
  getHistory: vi.fn(),
  checkIn: vi.fn(),
  draw: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/dailyLottery', () => ({
  default: {
    getStatus: mocks.getStatus,
    getHistory: mocks.getHistory,
    checkIn: mocks.checkIn,
    draw: mocks.draw
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError })
}))

vi.mock('@/utils/format', () => ({
  formatCurrency: (value: number) => `$${value.toFixed(2)}`,
  formatDateTimeToMinute: (value: string) => value
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'dailyLottery.won') return `Won ${params?.prize}`
        if (key === 'dailyLottery.rewardAdded') return `${params?.amount} added`
        return key
      }
    })
  }
})

const prizes = [
  { id: 'lucky', name: 'Lucky Prize', reward_amount: 1, weight: 1, enabled: true, probability: 0.5 },
  { id: 'thanks', name: 'Thanks', reward_amount: 0, weight: 1, enabled: true, probability: 0.5 }
]

const initialStatus = {
  enabled: true,
  checked_in: false,
  can_draw: false,
  already_drawn: false,
  today: '2026-09-02',
  timezone: 'Asia/Shanghai',
  next_reset_at: '2026-09-03T00:00:00+08:00',
  prizes
}

function mountView() {
  return mount(DailyLotteryView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: { template: '<span class="icon" />' }
      }
    }
  })
}

describe('DailyLotteryView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getStatus.mockResolvedValue(initialStatus)
    mocks.getHistory.mockResolvedValue([])
    mocks.checkIn.mockResolvedValue({ ...initialStatus, checked_in: true, can_draw: true })
    mocks.draw.mockResolvedValue({
      entry: {
        id: 1,
        user_id: 42,
        checkin_date: '2026-09-02',
        checked_in_at: '2026-09-02T08:00:00Z',
        drawn_at: '2026-09-02T08:01:00Z',
        prize_id: 'lucky',
        prize_name: 'Lucky Prize',
        reward_amount: 1,
        created_at: '2026-09-02T08:00:00Z',
        updated_at: '2026-09-02T08:01:00Z'
      },
      old_balance: 10,
      new_balance: 11
    })
  })

  it('requires check-in before drawing and renders the backend draw result', async () => {
    const wrapper = mountView()
    await flushPromises()

    const checkInButton = wrapper.get('[data-testid="daily-lottery-check-in"]')
    const drawButton = wrapper.get('[data-testid="daily-lottery-draw"]')
    expect(drawButton.attributes('disabled')).toBeDefined()

    await checkInButton.trigger('click')
    await flushPromises()
    expect(mocks.checkIn).toHaveBeenCalledTimes(1)
    expect(drawButton.attributes('disabled')).toBeUndefined()

    await drawButton.trigger('click')
    await flushPromises()

    expect(mocks.draw).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Won Lucky Prize')
    expect(wrapper.text()).toContain('$1.00 added')
    expect(wrapper.text()).toContain('dailyLottery.newBalance')
    expect(drawButton.attributes('disabled')).toBeDefined()
  })

  it('shows the disabled state without exposing action buttons', async () => {
    mocks.getStatus.mockResolvedValue({ ...initialStatus, enabled: false })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('dailyLottery.disabledTitle')
    expect(wrapper.find('[data-testid="daily-lottery-check-in"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="daily-lottery-draw"]').exists()).toBe(false)
  })
})
