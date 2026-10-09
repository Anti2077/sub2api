import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import ModelIdentityPanel from '../ModelIdentityPanel.vue'
import type { Account, Group } from '@/types'

const mocks = vi.hoisted(() => ({models:vi.fn(),config:vi.fn(),plans:vi.fn(),saveConfig:vi.fn(),savePlan:vi.fn(),updatePlan:vi.fn(),history:vi.fn(),list:vi.fn(),getById:vi.fn()}))
vi.mock('@/api/admin/modelIdentity',()=>mocks)
vi.mock('@/api/admin/users',()=>({list:mocks.list,getById:mocks.getById}))
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(key:string)=>key})}))
enableAutoUnmount(afterEach)
const account={id:1,name:'Target',platform:'openai',group_ids:[7,8]} as Account
const groups=[{id:7,name:'Billing',platform:'openai'},{id:8,name:'Incompatible',platform:'anthropic'}] as Group[]
const user={id:5,username:'tester',email:'test@example.com'}
function open(){return mount(ModelIdentityPanel,{props:{show:true,account,groups},global:{stubs:{BaseDialog:{template:'<div><slot /></div>'},Icon:true}}})}
describe('account model identity configuration',()=>{
  beforeEach(()=>{vi.resetAllMocks();mocks.models.mockResolvedValue({models:[{id:'openai/gpt-5.5',name:'GPT-5.5'}]});mocks.config.mockRejectedValue({response:{status:404}});mocks.plans.mockResolvedValue([]);mocks.list.mockResolvedValue({items:[user]});mocks.getById.mockResolvedValue(user);mocks.history.mockResolvedValue([])})
  it('saves selected user IDs and a compatible billing group',async()=>{
    mocks.saveConfig.mockResolvedValue({account_id:1,user_id:5,group_id:7,api_key_id:9,key_name:'Dedicated'})
    const w=open();await flushPromises();const selects=w.findAll('select');await selects[0].setValue(5);await selects[1].setValue(7)
    expect(selects[1].text()).not.toContain('Incompatible');await w.findAll('form')[0].trigger('submit');await flushPromises()
    expect(mocks.saveConfig).toHaveBeenCalledWith(1,{user_id:5,group_id:7});expect(w.text()).toContain('Dedicated')
  })
  it('creates schedules disabled with a 120-minute default and explicit model',async()=>{
    mocks.config.mockResolvedValue({account_id:1,user_id:5,group_id:7,api_key_id:9,key_name:'Dedicated'})
    const w=open();await flushPromises();const form=w.findAll('form')[1];await form.find('input').setValue('gpt-5.5');await form.find('select').setValue('openai/gpt-5.5');await form.trigger('submit');await flushPromises()
    expect(mocks.savePlan).toHaveBeenCalledWith({account_id:1,request_model:'gpt-5.5',expected_model:'openai/gpt-5.5',interval_minutes:120,enabled:false})
  })
  it('displays engine failures and disables plan creation',async()=>{
    mocks.models.mockRejectedValue(new Error('worker unavailable'));const w=open();await flushPromises();expect(w.get('[role="alert"]').text()).toContain('worker unavailable');expect(w.findAll('form')[1].get('button').attributes('disabled')).toBeDefined()
  })
  it('handles normalized API errors and missing configuration',async()=>{
    mocks.config.mockRejectedValue({status:404,message:'not configured'})
    const w=open();await flushPromises();expect(mocks.models).toHaveBeenCalled()
    mocks.saveConfig.mockRejectedValue({status:400,message:'configuration error: repair dedicated Key'})
    const selects=w.findAll('select');await selects[0].setValue(5);await selects[1].setValue(7)
    await w.findAll('form')[0].trigger('submit');await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('repair dedicated Key')
  })
})
