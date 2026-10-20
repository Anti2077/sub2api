import {flushPromises,mount} from '@vue/test-utils'
import {beforeEach,describe,it,expect,vi} from 'vitest'
import CardGrantManager from '../CardGrantManager.vue'
import ShopManager from '../ShopManager.vue'
import CodeManager from '../CodeManager.vue'
import CropManager from '../CropManager.vue'
import PrizeManager from '../PrizeManager.vue'
import {useManagementBatch} from '../useManagementBatch'
import type {CardTemplate,Shop,RewardCard} from '@/api/rewards'
const api=vi.hoisted(()=>({grant:vi.fn(),revoke:vi.fn(),shopStatus:vi.fn(),listCodes:vi.fn(),codes:vi.fn(),codeStatus:vi.fn()}))
vi.mock('@/api/rewards',()=>({rewardsAPI:api,rewardError:(e:Error)=>e.message}))
vi.mock('@/api/admin',()=>({adminAPI:{users:{list:vi.fn(),getById:vi.fn()}}}))
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string,v?:object)=>k+(v?JSON.stringify(v):'')})}))
const template:CardTemplate={id:1,version:2,name:'Luna',kind:'model_tokens',model:'luna',group_ids:[2],rate:.35,tokens:1000,hours:24,claim_days:30,published:true}
const global={stubs:{ConfirmDialog:{props:['show','message'],emits:['confirm','cancel'],template:'<div v-if="show" role="dialog">{{ message }}<slot/><button data-confirm @click="$emit(\'confirm\')">Confirm</button><button data-cancel @click="$emit(\'cancel\')">Cancel</button></div>'}}}
const button=(w:ReturnType<typeof mount>,key:string)=>w.findAll('button').find(b=>b.text()==='rewards.'+key)!
beforeEach(()=>{sessionStorage.clear();vi.clearAllMocks();api.grant.mockResolvedValue({});api.revoke.mockResolvedValue({});api.shopStatus.mockResolvedValue({});api.listCodes.mockResolvedValue([])})
describe('management actions',()=>{
 it('does not send a new grant when its retry ID cannot be persisted',async()=>{
  const w=mount(CardGrantManager,{props:{templates:[template],cards:[],groups:[],busy:false,reload:vi.fn()},global})
  await w.findAll('select')[1].setValue('1');await w.get('textarea').setValue('2');await w.get('input[maxlength="500"]').setValue('test');await w.get('form').trigger('submit')
  const storage=vi.spyOn(Storage.prototype,'setItem').mockImplementation(()=>{throw new Error('blocked')})
  await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.grant).not.toHaveBeenCalled();expect(w.get('[role=dialog]').text()).toContain('manageSessionFailed');storage.mockRestore();w.unmount()
 })
 it('restores uncertain grant and code batches after remount without creating new keys',async()=>{
  sessionStorage.setItem('rewards-management:session:grants',JSON.stringify([{id:3,template:1,key:'saved-grant-key',reason:'retry'}]))
  const props={templates:[template],cards:[],groups:[],busy:false,reload:vi.fn().mockResolvedValue(undefined)}
  let w=mount(CardGrantManager,{props,global});w.unmount();w=mount(CardGrantManager,{props,global})
  await button(w,'manageRetry').trigger('click');await w.get('[data-confirm]').trigger('click');await flushPromises()
  expect(api.grant).toHaveBeenCalledWith(3,1,'saved-grant-key','retry');expect(sessionStorage.getItem('rewards-management:session:grants')).toBeNull();w.unmount()
  sessionStorage.setItem('rewards-management:session:codes',JSON.stringify({template:1,count:2,days:7,key:'saved-code-key'}));api.codes.mockResolvedValue(['a','b'])
  const c=mount(CodeManager,{props:{templates:[template],groups:[],busy:false},global});await flushPromises();await button(c,'manageRetry').trigger('click');await c.get('[data-confirm]').trigger('click');await flushPromises()
  expect(api.codes).toHaveBeenCalledWith(1,2,7,'saved-code-key');expect(sessionStorage.getItem('rewards-management:session:codes')).toBeNull();c.unmount()
 })
 it('disables only unused codes and leaves used and expired codes intact',async()=>{
  api.listCodes.mockResolvedValue([{id:1,code:'one',status:'unused',template},{id:2,code:'two',status:'used',template},{id:3,code:'three',status:'unused',expires_at:'2000-01-01',template}])
  const w=mount(CodeManager,{props:{templates:[template],groups:[],busy:false},global});await flushPromises();await w.get('input[type=checkbox]').setValue(true);await button(w,'manageDisableCodes').trigger('click');await w.get('[data-confirm]').trigger('click');await flushPromises()
  expect(api.codeStatus).toHaveBeenCalledTimes(1);expect(api.codeStatus).toHaveBeenCalledWith(1,'disabled');w.unmount()
 })
 it('deduplicates recipients and retries only failures with identical request keys',async()=>{const w=mount(CardGrantManager,{props:{templates:[template],cards:[],groups:[],busy:false,reload:vi.fn().mockResolvedValue(undefined)},global});await w.findAll('select')[1].setValue('1');await w.get('textarea').setValue('2, 3 2');await w.get('input[maxlength="500"]').setValue('test');await w.get('form').trigger('submit');expect(api.grant).not.toHaveBeenCalled();api.grant.mockResolvedValueOnce({}).mockRejectedValueOnce(new Error('timeout'));await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.grant).toHaveBeenCalledTimes(2);const key=api.grant.mock.calls[1][2];await button(w,'manageRetry').trigger('click');await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.grant).toHaveBeenCalledTimes(3);expect(api.grant.mock.calls[2]).toEqual([3,1,key,'test'])})
 it('rejects malformed IDs and requires a revoke reason',async()=>{const card={id:10,user_id:2,template,status:'unused',source:'admin',remaining_tokens:1000,claim_expires_at:'2027-01-01'} as RewardCard;const w=mount(CardGrantManager,{props:{templates:[template],cards:[card],groups:[],busy:false,reload:vi.fn()},global});await w.findAll('select')[1].setValue('1');await w.get('textarea').setValue('2x');await w.get('input[maxlength="500"]').setValue('test');await w.get('form').trigger('submit');expect(w.get('[role=alert]').exists()).toBe(true);await button(w,'revoke').trigger('click');await w.get('[data-confirm]').trigger('click');expect(api.revoke).not.toHaveBeenCalled();await w.get('[role=dialog] input').setValue('review');await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.revoke).toHaveBeenCalledWith(10,'review')})
 it('changes shop status only and retains failed selections',async()=>{const item:Shop={id:5,version:9,template_id:1,template,price:20,stock:3,daily_limit:1,total_limit:2,published:true,starts_at:null,ends_at:null};const w=mount(ShopManager,{props:{items:[item],busy:false,reload:vi.fn()},global});await w.get('article input').setValue(true);api.shopStatus.mockRejectedValueOnce(new Error('conflict'));await button(w,'bulkUnpublish').trigger('click');await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.shopStatus).toHaveBeenCalledWith(5,9,false);expect((w.get('article input').element as HTMLInputElement).checked).toBe(true);expect(w.text()).toContain('conflict')})
 it('retries code generation with the same receipt key',async()=>{const w=mount(CodeManager,{props:{templates:[template],groups:[],busy:false},global});await flushPromises();await w.findAll('select')[1].setValue('1');await w.get('form').trigger('submit');api.codes.mockRejectedValueOnce(new Error('timeout')).mockResolvedValueOnce(['abc']);await w.get('[data-confirm]').trigger('click');await flushPromises();const args=api.codes.mock.calls[0];await button(w,'manageRetry').trigger('click');await w.get('[data-confirm]').trigger('click');await flushPromises();expect(api.codes.mock.calls[1]).toEqual(args);expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('abc')})
 it('keeps crop edits local until configuration is saved',async()=>{const crops=[{id:'one',name:'Seed',hours:1,yield:10,enabled:true,protection_minutes:30,steal_percent:10,max_stolen_percent:40}];const w=mount(CropManager,{props:{modelValue:crops,disabled:false}});await w.get('article input[type=checkbox]').setValue(true);await button(w,'manageDisable').trigger('click');expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject([{enabled:false}]);expect(crops[0].enabled).toBe(true)})
 it('computes prize probabilities and protects minimum prize count',async()=>{const prizes=[{id:'a',name:'A',reward_amount:1,weight:1,enabled:true},{id:'b',name:'B',reward_amount:0,weight:3,enabled:true}];const w=mount(PrizeManager,{props:{modelValue:prizes,templates:[template],disabled:false},global});expect(w.text()).toContain('25.00%');expect(w.text()).toContain('75.00%');await w.get('article input[type=checkbox]').setValue(true);expect(button(w,'managePrizeRemove').attributes('disabled')).toBeDefined()})
 it('locks through refresh and reports refresh errors while preserving receipts',async()=>{let release!:()=>void;const reload=vi.fn(()=>new Promise<void>(r=>{release=r})),busy=vi.fn(),save=vi.fn().mockResolvedValue({}),done=vi.fn(),b=useManagementBatch(reload,busy);const p=b.execute([{id:1}],v=>String(v.id),save,done);await flushPromises();await b.execute([{id:2}],v=>String(v.id),save,done);expect(save).toHaveBeenCalledTimes(1);expect(b.processing.value).toBe(true);release();await p;expect(b.processing.value).toBe(false);const failed=useManagementBatch(async()=>{throw new Error('offline')},vi.fn());await failed.execute([{id:3}],v=>String(v.id),save,done);expect(failed.results.value).toHaveLength(1);expect(failed.refreshError.value).toBe('offline')})
})
