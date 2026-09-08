import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TodaySpendCard from '../TodaySpendCard.vue'
vi.mock('vue-i18n', () => ({useI18n:()=>({t:(key:string)=>key})}))

describe('TodaySpendCard', () => {
  it('does not add user charges to upstream cost, or abbreviate large monetary totals', () => {
    const wrapper=mount(TodaySpendCard,{props:{actualCost:12345.6789,accountCost:4567.8912}})
    expect(wrapper.get('[data-testid="today-actual-spend"] dd').text()).toBe('$12,345.6789')
    expect(wrapper.get('[data-testid="today-account-spend"] dd').text()).toBe('$4,567.8912')
  })
  it('distinguishes an unavailable cost field from a confirmed zero', () => {
    const wrapper=mount(TodaySpendCard,{props:{actualCost:0,accountCost:undefined}})
    expect(wrapper.get('[data-testid="today-actual-spend"] dd').text()).toBe('$0.00')
    expect(wrapper.get('[data-testid="today-account-spend"] dd').text()).toBe('—')
  })
})
