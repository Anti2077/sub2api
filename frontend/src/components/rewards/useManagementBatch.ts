import { ref } from 'vue'
import { rewardError } from '@/api/rewards'
export function useManagementBatch(reload: () => Promise<void>, setBusy: (value: boolean) => void) {
  const processing=ref(false), results=ref<{id:number|string;name:string;error?:string}[]>([]), refreshError=ref('')
  async function execute<T extends {id:number|string}>(rows:T[], label:(row:T)=>string, save:(row:T)=>Promise<unknown>, success:(id:number|string)=>void) {
    if(processing.value)return
    processing.value=true;setBusy(true);results.value=[];refreshError.value=''
    try {
      for(const row of rows) {
        try {await save(row);success(row.id);results.value.push({id:row.id,name:label(row)})}
        catch(e){results.value.push({id:row.id,name:label(row),error:rewardError(e)})}
      }
      try{await reload()}catch(e){refreshError.value=rewardError(e)}
    }finally{processing.value=false;setBusy(false)}
  }
  return {processing,results,refreshError,execute}
}
