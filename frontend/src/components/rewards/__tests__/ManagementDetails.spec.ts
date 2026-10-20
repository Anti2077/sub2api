import {mount} from '@vue/test-utils'
import {describe,it,expect,vi} from 'vitest'
import TemplatePicker from '../TemplatePicker.vue'
import RecordsManager from '../RecordsManager.vue'
import type {CardTemplate} from '@/api/rewards'
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string)=>k,te:(k:string)=>k==='rewards.manageEvents.revoked'})}))
const templates:CardTemplate[]=[{id:1,version:2,name:'Luna',kind:'model_tokens',model:'luna',group_ids:[3],rate:.35,tokens:1000,hours:24,claim_days:30,published:true},{id:2,version:1,name:'Draft',kind:'group_rate',model:'',group_ids:[3],rate:.25,tokens:0,hours:24,claim_days:30,published:false}]
describe('management details',()=>{
 it('searches published templates by group and preserves a selected draft',async()=>{
  const w=mount(TemplatePicker,{props:{modelValue:2,templates,groups:[{id:3,name:'GPT team'}]}})
  await w.get('input').setValue('GPT team');expect(w.findAll('option').some(v=>v.text().includes('Luna'))).toBe(true)
  await w.get('input').setValue('missing');expect(w.findAll('option').some(v=>v.text().includes('Draft'))).toBe(true)
  expect(w.findAll('option').some(v=>v.text().includes('Luna'))).toBe(false);expect(w.get('[role=alert]').text()).toContain('manageUnpublished');expect(w.emitted('update:modelValue')).toBeUndefined()
 })
 it('keeps audit records read-only with translated kinds and raw searchable details',async()=>{
  const w=mount(RecordsManager,{props:{title:'Audit',items:[{id:1,kind:'revoked',created_at:'2026-10-19',user_id:2,detail:{reason:'review'}},{id:2,kind:'future_kind',created_at:'2026-10-19'}]}})
  expect(w.find('input[type=checkbox]').exists()).toBe(false);expect(w.text()).toContain('rewards.manageEvents.revoked');expect(w.text()).toContain('future_kind')
  await w.get('input[type=search]').setValue('review');expect(w.findAll('article')).toHaveLength(1);expect(w.text()).toContain('manageDetails')
 })
})
