import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TemplateFields from '../TemplateFields.vue'
import SearchMultiSelect from '@/components/common/SearchMultiSelect.vue'
import type { CardTemplate } from '@/api/rewards'
import { templateModels } from '@/components/rewards/templateValidation'
const api=vi.hoisted(()=>({getModelAllowlistCandidates:vi.fn()}))
vi.mock('@/api/admin',()=>({adminAPI:{groups:api}}))
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string)=>k})}))
function render(overrides:Partial<CardTemplate>={}){
 const value=reactive<CardTemplate>({id:1,version:1,name:'Test',kind:'model_day',model:'legacy',group_ids:[1],rate:.5,tokens:100,hours:24,claim_days:30,published:false,...overrides})
 const w=mount(TemplateFields,{props:{modelValue:value,groups:[{id:1,name:'A'},{id:2,name:'B'}],disabled:false}})
 return {w,value}
}
beforeEach(()=>{api.getModelAllowlistCandidates.mockReset().mockResolvedValue(['gpt-a','gpt-b','gpt-*'])})
describe('reward template editor',()=>{
 it('loads searchable multi-model candidates and preserves legacy selections across groups',async()=>{
  const {w,value}=render();await flushPromises()
  const models=w.findAllComponents(SearchMultiSelect)[1]
  expect(models.props('modelValue')).toEqual(['legacy'])
  expect(models.props('options').map(v=>v.value)).toEqual(['gpt-a','gpt-b','legacy'])
  models.vm.$emit('update:modelValue',['gpt-a','gpt-b']);await flushPromises()
  expect(templateModels(value)).toEqual(['gpt-a','gpt-b'])
  value.group_ids=[2];await flushPromises();expect(api.getModelAllowlistCandidates).toHaveBeenLastCalledWith(2)
  expect(templateModels(value)).toEqual(['gpt-a','gpt-b']);w.unmount()
 })
 it('adds and deduplicates pasted IDs, and includes pending input when saving',async()=>{
  const {w,value}=render();await w.get('textarea').setValue('gpt-a, gpt-b\ngpt-a')
  expect(w.vm.prepare()).toBe(true);expect(templateModels(value)).toEqual(['legacy','gpt-a','gpt-b'])
  await w.get('textarea').setValue('gpt-*');expect(w.vm.prepare()).toBe(false);await flushPromises()
  expect(w.get('[role=alert]').text()).toBe('rewards.invalidModels');expect(templateModels(value)).toHaveLength(3);w.unmount()
 })
 it('edits multi-day durations and keeps short token durations compatible',async()=>{
  const {w,value}=render({hours:7*24});expect((w.get('input[type=number]').element as HTMLInputElement).value).toBe('7')
  await w.get('input[type=number]').setValue(3);expect(value.hours).toBe(72);w.unmount()
  const token=render({kind:'model_tokens',hours:5});expect((token.w.get('select[aria-label="rewards.durationUnit"]').element as HTMLSelectElement).value).toBe('hours')
  token.value.kind='model_day';await flushPromises();expect(token.value.hours).toBe(24);token.w.unmount()
 })
 it('allows manual IDs after candidate failure and disables controls during saving',async()=>{
  api.getModelAllowlistCandidates.mockRejectedValue(new Error('offline'));const {w,value}=render();await flushPromises()
  expect(w.text()).toContain('rewards.modelsLoadFailed');await w.get('textarea').setValue('new-model');expect(w.vm.prepare()).toBe(true)
  expect(templateModels(value)).toContain('new-model');await w.setProps({disabled:true})
  expect(w.get('textarea').attributes('disabled')).toBeDefined();expect(w.findAllComponents(SearchMultiSelect).every(v=>v.props('disabled'))).toBe(true);w.unmount()
 })
})
