import { reactive } from 'vue';
const user=reactive({balance:128,concurrency:4});
export const useAuthStore=()=>({user,refreshUser:async()=>user});
export const useSubscriptionStore=()=>({fetchActiveSubscriptions:async()=>[]});
export const useAppStore=()=>({showError:console.error,showWarning:console.warn,showSuccess:()=>{}});
export const authAPI={getPublicSettings:async()=>({})};
export const redeemAPI={getHistory:async()=>({items:[],total:0}),redeem:async code=>{
 await new Promise(r=>setTimeout(r,400));
 if(code==='FAIL')throw {response:{data:{reason:'REDEEM_CODE_NOT_FOUND',detail:'演示：兑换码无效'}}};
 if(code==='CONCURRENCY'){user.concurrency+=2;return{type:'concurrency',value:2,new_concurrency:user.concurrency,message:'并发额度已增加'}};
 if(code==='SUBSCRIPTION')return{type:'subscription',value:30,group_name:'演示订阅',validity_days:30,message:'订阅已开通'};
 user.balance+=20;return{type:'balance',value:20,new_balance:user.balance,message:'兑换成功，额度已到账'};
}};