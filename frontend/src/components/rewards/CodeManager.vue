<template>
  <section class="space-y-5">
    <form class="card space-y-4 p-5" @submit.prevent="prepareGenerate"><fieldset :disabled="locked || !!job" class="space-y-4"><h2 class="text-lg font-semibold">{{ t('rewards.codes') }}</h2><TemplatePicker v-model="templateID" :templates="templates" :groups="groups" /><div class="grid gap-3 sm:grid-cols-2"><label>{{ t('rewards.count') }}<input v-model.number="count" class="input mt-1" type="number" min="1" max="100" required></label><label>{{ t('rewards.days') }}<input v-model.number="days" class="input mt-1" type="number" min="1" max="365" required></label></div><button class="btn btn-primary min-h-11" :disabled="!templates.some(v=>v.id===templateID && v.published)">{{ t('rewards.generate') }}</button></fieldset></form>
    <div v-if="job" class="card space-y-3 p-4"><p class="text-sm">{{ t('rewards.manageCodeRetry') }}</p><div class="flex flex-wrap gap-2"><button class="btn btn-primary min-h-11" :disabled="locked" @click="pending='generate'">{{ t('rewards.manageRetry') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked" @click="pending='discard'">{{ t('rewards.manageEndBatch') }}</button></div></div>
    <p v-if="error" role="alert" class="break-words text-red-700 dark:text-red-300">{{ error }}</p>
    <div v-if="generated.length" class="card space-y-3 p-4"><h3 class="font-semibold">{{ t('rewards.manageGenerated',{count:generated.length}) }}</h3><textarea class="input min-h-32 font-mono" readonly :aria-label="t('rewards.manageGeneratedCodes')" :value="generated.join('\n')" /><button class="btn btn-secondary min-h-11" @click="copy(generated)">{{ t('rewards.manageCopy') }}</button></div>
    <BatchResults :results="results" :refresh-error="refreshError" />
    <p v-if="notice" role="status" class="text-sm">{{ notice }}</p>
    <ManagementList v-model="selected" :items="rows" :title="t('rewards.manageCodeList')" :busy="locked" :note="t('rewards.manageRecentCodes')" :search-text="v=>`${v.id} ${v.code} ${v.template.name} ${templateModels(v.template).join(', ')} ${v.used_by||''}`" :row-label="v=>`#${v.id}`" :filters="filters" :filter-value="(v,k)=>k==='kind'?v.template.kind:status(v)" :selectable="()=>true">
      <template #header><button class="btn btn-secondary min-h-11" :disabled="locked" @click="refresh">{{ t('rewards.refresh') }}</button></template>
      <template #actions="{selected:chosen}"><button class="btn btn-secondary min-h-11" :disabled="locked || !chosen.length" @click="copy(chosen.map(v=>v.code))">{{ t('rewards.manageCopySelected') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked || !chosen.some(v=>status(v)==='unused')" @click="pending='disable'">{{ t('rewards.manageDisableCodes') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked || !chosen.some(v=>status(v)==='disabled')" @click="pending='enable'">{{ t('rewards.manageEnableCodes') }}</button></template>
      <template #default="{row}"><div class="space-y-1 text-sm"><h3 class="break-words font-semibold">#{{ row.id }} · {{ row.template.name }}</h3><p class="break-all font-mono">{{ row.code }}</p><p>{{ t(`rewards.${row.template.kind}`) }} · {{ statusLabel(status(row)) }}</p><p>{{ t('rewards.ends') }}: {{ row.expires_at ? new Date(row.expires_at).toLocaleString() : '—' }} · {{ t('rewards.userID') }} #{{ row.used_by||'—' }}</p></div></template>
    </ManagementList>
    <ConfirmDialog :show="!!pending" :title="t(pending==='discard'?'rewards.manageEndBatch':pending==='generate'?'rewards.generate':pending==='disable'?'rewards.manageDisableCodes':'rewards.manageEnableCodes')" :message="pending==='discard'?t('rewards.manageDiscardCodes'):pending==='generate'?t('rewards.manageGenerateConfirm',{count:job?.count||0,days:job?.days||0}):t('rewards.manageCodeStatusConfirm',{count:changes.length})" @cancel="cancel" @confirm="confirm" />
  </section>
</template>
<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import {computed,onMounted,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {rewardsAPI,rewardError,type CardTemplate,type RewardCode} from '@/api/rewards'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ManagementList from './ManagementList.vue'
import TemplatePicker from './TemplatePicker.vue'
import BatchResults from './BatchResults.vue'
import {useManagementBatch} from './useManagementBatch'
import {readManagementSession,writeManagementSession} from './managementSession'
const props=defineProps<{templates:CardTemplate[];groups:{id:number;name:string}[];busy:boolean}>(),emit=defineEmits<{busy:[value:boolean]}>(),{t}=useI18n()
const rows=ref<RewardCode[]>([]),selected=ref<(number|string)[]>([]),templateID=ref(0),count=ref(1),days=ref(30),generated=ref<string[]>([]),pending=ref(''),error=ref(''),notice=ref(''),working=ref(false),attempted=ref(false)
type CodeJob={template:number;count:number;days:number;key:string}
const savedJob=readManagementSession<CodeJob>('codes',(v):v is CodeJob=>{if(!v||typeof v!=='object')return false;const j=v as CodeJob;return Number.isSafeInteger(j.template)&&j.template>0&&Number.isInteger(j.count)&&j.count>=1&&j.count<=100&&Number.isInteger(j.days)&&j.days>=1&&j.days<=365&&typeof j.key==='string'&&j.key.length>=8})
const job=ref<CodeJob|undefined>(savedJob)
attempted.value=!!savedJob
const {processing,results,refreshError,execute}=useManagementBatch(load,v=>emit('busy',v)),locked=computed(()=>props.busy||processing.value||working.value)
function status(v:RewardCode){return v.status==='used'?'used':v.expires_at && new Date(v.expires_at).getTime()<=Date.now()?'expired':v.status}
function statusLabel(v:string){return v==='used'?t('rewards.manageCodeUsed'):v==='unused'?t('rewards.manageCodeUnused'):v==='disabled'?t('rewards.manageDisabled'):t(`rewards.${v}`)}
const filters=computed(()=>[{key:'kind',label:t('rewards.kind'),options:['model_day','group_rate','model_tokens'].map(value=>({value,label:t(`rewards.${value}`)}))},{key:'status',label:t('rewards.manageStatus'),options:['unused','used','disabled','expired'].map(value=>({value,label:statusLabel(value)}))}])
const changes=computed(()=>rows.value.filter(v=>selected.value.includes(v.id)&&status(v)===(pending.value==='disable'?'unused':'disabled')))
async function load(){rows.value=await rewardsAPI.listCodes()}
async function refresh(){if(locked.value)return;working.value=true;emit('busy',true);try{error.value='';await load()}catch(e){error.value=rewardError(e)}finally{working.value=false;emit('busy',false)}}
function prepareGenerate(){if(locked.value||job.value)return;job.value={template:templateID.value,count:count.value,days:days.value,key:crypto.randomUUID()};attempted.value=false;pending.value='generate'}
function cancel(){if(pending.value==='generate'&&!attempted.value)job.value=undefined;pending.value=''}
async function copy(values:string[]){try{await navigator.clipboard.writeText(values.join('\n'));notice.value=t('rewards.manageCopied',{count:values.length})}catch{error.value=t('rewards.manageCopyFailed')}}
async function confirm(){
 if(locked.value)return
 if(pending.value==='discard'){try{writeManagementSession('codes');job.value=undefined;pending.value=''}catch{error.value=t('rewards.manageSessionFailed')}return}
 if(pending.value==='generate'){if(!job.value)return;const v={...job.value};try{writeManagementSession('codes',v)}catch{error.value=t('rewards.manageSessionFailed');return}pending.value='';working.value=true;emit('busy',true);error.value='';attempted.value=true;try{generated.value=await rewardsAPI.codes(v.template,v.count,v.days,v.key);writeManagementSession('codes');job.value=undefined;await load()}catch(e){error.value=rewardError(e)}finally{working.value=false;emit('busy',false)}return}
 const target=pending.value==='disable'?'disabled':'unused',batch=changes.value.map(v=>({...v}));pending.value=''
 await execute(batch,v=>`#${v.id}`,v=>rewardsAPI.codeStatus(v.id,target),id=>{selected.value=selected.value.filter(v=>v!==id)})
}
onMounted(refresh)
</script>
