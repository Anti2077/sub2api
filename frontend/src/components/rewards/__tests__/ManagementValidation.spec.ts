import {flushPromises,mount} from '@vue/test-utils'
import {describe,it,expect,vi} from 'vitest'
import {invalidCrops,invalidPrizes,validPrizePool} from '../managementValidation'
import CropManager from '../CropManager.vue'
import type {Crop,CardTemplate} from '@/api/rewards'
import type {DailyLotteryPrize} from '@/api/dailyLottery'
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string,v?:object)=>k+(v?JSON.stringify(v):'')})}))
const crop=(id:string):Crop=>({id,name:id,hours:1,yield:10,enabled:true,protection_minutes:30,steal_percent:10,max_stolen_percent:40})
const template={id:1,published:true} as CardTemplate
const prize=(id:string):DailyLotteryPrize=>({id,name:id,weight:1,reward_amount:0,enabled:true})
describe('management form validation',()=>{
 it('validates hidden crop values, integer limits, UTF-8 names and duplicate IDs',()=>{
  expect(invalidCrops([crop('a')])).toEqual([])
  const values=[{...crop('a'),hours:1.5},{...crop('b'),yield:NaN},{...crop('c'),name:'菜'.repeat(41)},{...crop('d'),hours:0},{...crop('e'),yield:1000001}]
  expect(invalidCrops(values)).toHaveLength(5);expect(invalidCrops([crop('a'),crop('a')])).toHaveLength(2)
 })
 it('enforces valid prize rows and pool constraints independently of visible controls',()=>{
  expect(validPrizePool([prize('a'),prize('b')],[template])).toBe(true)
  expect(validPrizePool([prize('a')],[template])).toBe(false)
  expect(validPrizePool([{...prize('a'),weight:600000000},{...prize('b'),weight:600000000}],[template])).toBe(false)
  expect(validPrizePool([{...prize('a'),enabled:false},{...prize('b'),enabled:false}],[template])).toBe(false)
  expect(invalidPrizes([{...prize('a'),card_template_id:-1},{...prize('b'),card_template_id:1,reward_amount:1},{...prize('c'),weight:0.1}], [template])).toHaveLength(3)
  expect(invalidPrizes([{...prize('a'),card_template_id:1}], [{...template,published:false}])).toHaveLength(1)
  expect(invalidPrizes([{...prize('a'),card_template_id:1}], [template])).toHaveLength(0)
 })
 it('reveals an invalid crop on a different page after clearing active filters',async()=>{
  const rows=Array.from({length:15},(_,i)=>crop('crop'+i));rows[14].hours=0
  const w=mount(CropManager,{props:{modelValue:rows,disabled:false}});await w.get('input[type=search]').setValue('crop1');await w.get('[role=alert] button').trigger('click');await flushPromises()
  expect((w.get('input[type=search]').element as HTMLInputElement).value).toBe('');expect(w.findAll('article')).toHaveLength(3);expect(w.get('[data-row-id="crop14"]')).toBeDefined();w.unmount()
 })
})
