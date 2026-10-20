import {flushPromises,mount} from '@vue/test-utils'
import {describe,it,expect,vi} from 'vitest'
import SearchMultiSelect from '@/components/common/SearchMultiSelect.vue'
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string)=>k})}))
const props={modelValue:[99],options:[{value:1,label:'GPT A'},{value:2,label:'GPT B'},{value:3,label:'Luna'}],label:'Models'}
describe('reward rule multiselect',()=>{
 it('selects only filtered options, preserves unavailable selections, and removes chips independently',async()=>{
  const w=mount(SearchMultiSelect,{props});expect(w.find('button button').exists()).toBe(false);await w.get('button[aria-haspopup]').trigger('click');await w.get('input').setValue('GPT')
  const select=w.findAll('button').find(v=>v.text()==='common.selectAll (2)')!;await select.trigger('click');expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([99,1,2]);await w.setProps({modelValue:[99,1,2]})
  expect(w.findAll('[role=option]')).toHaveLength(2);expect(w.findAll('[role=option]').every(v=>v.attributes('aria-selected')==='true')).toBe(true)
  await w.get('button[aria-label="Remove option: GPT A"]').trigger('click');expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([99,2]);await w.findAll('button').find(v=>v.text()==='common.clear')!.trigger('click');expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([]);w.unmount()
 })
 it('closes and disables every control during saves, and supports escape',async()=>{
  const w=mount(SearchMultiSelect,{props});await w.get('button[aria-haspopup]').trigger('click');await flushPromises();await w.get('input').trigger('keydown.esc');expect(w.find('[role=listbox]').exists()).toBe(false);await w.get('button[aria-haspopup]').trigger('click');await w.setProps({disabled:true});expect(w.find('[role=listbox]').exists()).toBe(false);expect(w.findAll('button').every(v=>v.attributes('disabled')!==undefined)).toBe(true);w.unmount()
 })
})
