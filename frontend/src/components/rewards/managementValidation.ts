import type {DailyLotteryPrize} from '@/api/dailyLottery'
import type {CardTemplate,Crop} from '@/api/rewards'
const finite=(v:number)=>typeof v==='number'&&Number.isFinite(v)
const integer=(v:number,min:number,max:number)=>finite(v)&&Number.isSafeInteger(v)&&v>=min&&v<=max
export function invalidCrops(rows:Crop[]):Crop[]{
  const counts=new Map<string,number>();for(const row of rows)counts.set(row.id,(counts.get(row.id)||0)+1)
  return rows.filter(v=>!v.id||v.id.length>60||(counts.get(v.id)||0)>1||!v.name.trim()||new TextEncoder().encode(v.name).length>120||!integer(v.hours,1,720)||!integer(v.yield,1,1000000))
}
export function invalidPrizes(rows:DailyLotteryPrize[],templates:CardTemplate[]):DailyLotteryPrize[]{
  const counts=new Map<string,number>();for(const row of rows)counts.set(row.id,(counts.get(row.id)||0)+1)
  return rows.filter(v=>!v.id||v.id.length>64||(counts.get(v.id)||0)>1||!v.name.trim()||[...v.name].length>40||!integer(v.weight,1,1000000000)||!finite(v.reward_amount)||v.reward_amount<0||v.reward_amount>1000000||(v.card_template_id!==undefined&&(!integer(v.card_template_id,0,Number.MAX_SAFE_INTEGER)||v.card_template_id>0&&(v.reward_amount!==0||!templates.some(t=>t.id===v.card_template_id&&t.published)))))
}
export function validPrizePool(rows:DailyLotteryPrize[],templates:CardTemplate[]):boolean{
  return rows.length>=2&&rows.length<=8&&rows.some(v=>v.enabled)&&rows.reduce((n,v)=>n+v.weight,0)<=1000000000&&!invalidPrizes(rows,templates).length
}
