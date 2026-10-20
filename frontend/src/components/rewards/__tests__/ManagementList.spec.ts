import {mount} from '@vue/test-utils'
import {describe,it,expect,vi} from 'vitest'
import ManagementList from '../ManagementList.vue'
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string,v?:object)=>k+JSON.stringify(v||{})})}))
const items=Array.from({length:15},(_,i)=>({id:i+1,name:'row '+(i+1),enabled:i!==0,kind:i%2?'card':'balance'}))
function render(){return mount(ManagementList,{props:{items,title:'List',searchText:(v:{name:string})=>v.name,rowLabel:(v:{name:string})=>v.name,selectable:(v:{enabled:boolean})=>v.enabled,modelValue:[],filters:[{key:'kind',label:'Kind',options:[{value:'card',label:'Card'}]}],filterValue:(v:{kind:string})=>v.kind}})}
describe('ManagementList',()=>{
 it('selects all eligible matches across pages, inverts only this page, and keeps selection when resizing',async()=>{
  const w=render();const action=(key:string)=>w.findAll('button').find(b=>b.text().startsWith('rewards.'+key))!
  await action('manageSelectMatching').trigger('click');const all=Array.from({length:14},(_,i)=>i+2);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(all);await w.setProps({modelValue:all})
  await action('manageInvertPage').trigger('click');expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([13,14,15])
  await w.get('footer select').setValue('24');expect(w.findAll('article')).toHaveLength(15);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([13,14,15]);w.unmount()
 })
 it('resets filters explicitly and gives read-only records no selection controls',async()=>{
  const w=render();await w.get('input[type=search]').setValue('missing');expect(w.findAll('article')).toHaveLength(0);await w.findAll('button').find(b=>b.text().startsWith('rewards.manageResetFilters'))!.trigger('click');expect(w.findAll('article')).toHaveLength(12)
  await w.setProps({selectable:undefined});expect(w.find('input[type=checkbox]').exists()).toBe(false);expect(w.text()).not.toContain('manageSelectMatching');w.unmount()
 })
 it('selects eligible page rows and preserves other pages',async()=>{const w=render();await w.get('input[type=checkbox]').setValue(true);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(Array.from({length:11},(_,i)=>i+2));await w.setProps({modelValue:Array.from({length:11},(_,i)=>i+2)});await w.findAll('footer button')[1].trigger('click');expect(w.findAll('article')).toHaveLength(3);expect((w.get('input[type=checkbox]').element as HTMLInputElement).checked).toBe(false);await w.get('article input').setValue(true);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toContain(13)})
 it('resets selection and page for searches or filters',async()=>{const w=render();await w.setProps({modelValue:[3]});await w.findAll('footer button')[1].trigger('click');await w.get('select').setValue('card');expect(w.findAll('article')).toHaveLength(7);expect(w.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([]);await w.get('input[type=search]').setValue('row 14');expect(w.findAll('article')).toHaveLength(1)})
 it('disables controls while busy and never submits enclosing forms',async()=>{const w=render();await w.setProps({busy:true});expect(w.findAll('button').every(b=>b.attributes('type')==='button')).toBe(true);expect(w.get('select').attributes('disabled')).toBeDefined();expect(w.get('input[type=checkbox]').attributes('disabled')).toBeDefined()})
})
