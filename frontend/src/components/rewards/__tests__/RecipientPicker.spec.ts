import {flushPromises,mount} from '@vue/test-utils'
import {beforeEach,describe,it,expect,vi} from 'vitest'
import RecipientPicker from '../RecipientPicker.vue'
const users=vi.hoisted(()=>({list:vi.fn(),getById:vi.fn()}))
vi.mock('@/api/admin',()=>({adminAPI:{users}}))
vi.mock('@/api/rewards',()=>({rewardError:(e:Error)=>e.message}))
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string,v?:object)=>k+(v?JSON.stringify(v):'')})}))
const user=(id:number)=>({id,username:`User ${id}`,email:`user${id}@test.invalid`,status:'active'})
beforeEach(()=>{vi.clearAllMocks();users.list.mockResolvedValue({items:[user(1),user(2)],total:15});users.getById.mockResolvedValue(user(7))})
async function open(w:ReturnType<typeof mount>){w.get('details').element.setAttribute('open','');await w.get('details').trigger('toggle');await flushPromises()}
describe('RecipientPicker',()=>{
 it('loads lazily and preserves recipients across server pages and searches',async()=>{
  const w=mount(RecipientPicker,{props:{modelValue:[90],disabled:false}});expect(users.list).not.toHaveBeenCalled();await open(w)
  expect(users.list).toHaveBeenCalledWith(1,12,{search:'',status:'active'},expect.objectContaining({signal:expect.any(AbortSignal)}))
  await w.get('input[type=checkbox]').setValue(true);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([90,1,2]);await w.setProps({modelValue:[90,1,2]})
  users.list.mockResolvedValue({items:[user(13)],total:15});await w.findAll('footer button')[1].trigger('click');await flushPromises();expect(users.list.mock.calls.at(-1)?.[0]).toBe(2)
  await w.get('input[type=checkbox]').setValue(true);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([90,1,2,13])
  await w.get('input[type=search]').setValue('user13');await w.get('input[type=search]').trigger('keydown.enter');await flushPromises();expect(users.list.mock.calls.at(-1)?.slice(0,3)).toEqual([1,12,{search:'user13',status:'active'}]);w.unmount()
 })
 it('uses exact lookup for numeric IDs and treats missing users as an empty result',async()=>{
  const w=mount(RecipientPicker,{props:{modelValue:[],disabled:false}});await open(w);await w.get('input[type=search]').setValue('7');await w.get('input[type=search]').trigger('keydown.enter');await flushPromises();expect(users.getById).toHaveBeenCalledWith(7);expect(w.text()).toContain('User 7')
  users.getById.mockRejectedValue({response:{status:404}});await w.get('input[type=search]').setValue('99');await w.get('input[type=search]').trigger('keydown.enter');await flushPromises();expect(w.text()).toContain('manageRecipientEmpty');expect(w.find('[role=alert]').exists()).toBe(false);w.unmount()
 })
 it('enforces the 100-user boundary without silently truncating and disables mutations while locked',async()=>{
  const w=mount(RecipientPicker,{props:{modelValue:Array.from({length:100},(_,i)=>i+10),disabled:false}});await open(w);await w.get('input[type=checkbox]').setValue(true);expect(w.emitted('update:modelValue')).toBeUndefined();expect(w.get('[role=alert]').text()).toContain('manageRecipientLimit');await w.setProps({disabled:true});expect(w.findAll('button').every(v=>v.attributes('disabled')!==undefined)).toBe(true);w.unmount()
 })
 it('keeps request failures visible and allows a successful retry',async()=>{
  users.list.mockRejectedValueOnce(new Error('offline'));const w=mount(RecipientPicker,{props:{modelValue:[],disabled:false}});await open(w);expect(w.get('[role=alert]').text()).toBe('offline');await w.findAll('button')[0].trigger('click');await flushPromises();expect(w.find('[role=alert]').exists()).toBe(false);expect(w.text()).toContain('User 1');w.unmount()
 })
})
