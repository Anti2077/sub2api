import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DailyLotteryView from '../DailyLotteryView.vue'

const mocks = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  getHistory: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dailyLottery: {
      getConfig: mocks.getConfig,
      updateConfig: mocks.updateConfig,
      getHistory: mocks.getHistory
    }
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
    useI18n: () => ({ locale: { value: 'en-US' }, t: (key: string) => key })
  }
})

const config = {
  enabled: false,
  prizes: [
    { id: 'first', name: 'First Prize', reward_amount: 1, weight: 1, enabled: true },
    { id: 'thanks', name: 'Thanks', reward_amount: 0, weight: 1, enabled: true }
  ]
}

function mountView() {
  return mount(DailyLotteryView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: { template: '<span class="icon" />' },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<button type="button" class="toggle-stub" @click="$emit(\'update:modelValue\', !modelValue)" />'
        },
        Pagination: { template: '<div class="pagination-stub" />' }
      }
    }
  })
}

describe('Admin DailyLotteryView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getConfig.mockResolvedValue(config)
    mocks.getHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })
    mocks.updateConfig.mockImplementation(async (value) => ({
      ...value,
      prizes: value.prizes.map((prize: { id: string }, index: number) => ({ ...prize, id: prize.id || `generated-${index}` }))
    }))
  })

  it('adds a valid prize level and saves the complete configuration', async () => {
    const wrapper = mountView()
    await flushPromises()

    const addButton = wrapper.findAll('button').find((button) => button.text().includes('dailyLottery.admin.addPrize'))
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')

    const nameInputs = wrapper.findAll('input[type="text"]')
    const rewardInputs = wrapper.findAll('input[step="0.00000001"]')
    const weightInputs = wrapper.findAll('input[step="1"]')
    expect(nameInputs).toHaveLength(3)

    await nameInputs[2].setValue('Lucky Prize')
    await rewardInputs[2].setValue('0.25')
    await weightInputs[2].setValue('8')
    await flushPromises()

    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('dailyLottery.admin.save'))
    expect(saveButton).toBeTruthy()
    expect(saveButton!.attributes('disabled')).toBeUndefined()
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.updateConfig).toHaveBeenCalledTimes(1)
    expect(mocks.updateConfig.mock.calls[0][0].prizes).toHaveLength(3)
    expect(mocks.updateConfig.mock.calls[0][0].prizes[2]).toMatchObject({
      name: 'Lucky Prize',
      reward_amount: 0.25,
      weight: 8,
      enabled: true
    })
  })

  it('keeps the minimum two prize levels from being deleted', async () => {
    const wrapper = mountView()
    await flushPromises()

    const removeButtons = wrapper.findAll('button[aria-label^="dailyLottery.admin.removePrize"]')
    expect(removeButtons).toHaveLength(2)
    expect(removeButtons.every((button) => button.attributes('disabled') !== undefined)).toBe(true)
  })
})
