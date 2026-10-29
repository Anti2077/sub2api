<template>
  <section class="space-y-5">
    <form class="card space-y-4 p-5" @submit.prevent="prepareGrant"><fieldset :disabled="locked || grantJobs.length>0" class="space-y-4">
      <h2 class="text-lg font-semibold">{{ t('rewards.manageBulkGrant') }}</h2>
      <TemplatePicker v-model="templateID" :templates="templates" :groups="groups" />
      <RecipientPicker :model-value="recipientIDs" :disabled="locked || grantJobs.length>0" @update:model-value="recipientText=$event.join(', ')" />
      <label class="block">{{ t('rewards.manageRecipients') }}<textarea v-model="recipientText" class="input mt-1 min-h-24" :aria-label="t('rewards.manageRecipients')" required /></label>
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('rewards.manageRecipientsHint') }}</p>
      <label class="block">{{ t('rewards.reason') }}<input v-model="grantReason" class="input mt-1" required maxlength="500"></label>
      <button class="btn btn-primary min-h-11" :disabled="!templates.some(v=>v.id===templateID && v.published)">{{ t('rewards.manageReviewGrant') }}</button>
    </fieldset></form>
    <p v-if="validation" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ validation }}</p>
    <div v-if="grantJobs.length" class="card space-y-3 p-4"><p class="text-sm">{{ t('rewards.manageGrantRetry',{count:grantJobs.length}) }}</p><div class="flex flex-wrap gap-2"><button class="btn btn-primary min-h-11" :disabled="locked" @click="pending='grant'">{{ t('rewards.manageRetry') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked" @click="pending='discard'">{{ t('rewards.manageEndBatch') }}</button></div></div>
    <BatchResults :results="results" :refresh-error="refreshError" />
    <ManagementList v-model="selected" :items="cards" :title="t('rewards.manageIssued')" :busy="locked" :note="t('rewards.manageRecentCards')" :search-text="cardSearch" :row-label="v=>`#${v.id} · ${v.template.name}`" :filters="filters" :filter-value="cardFilter" :selectable="v=>v.status!=='revoked'">
      <template #actions><button class="btn btn-secondary min-h-11" :disabled="locked || !selected.length" @click="prepareRevoke()">{{ t('rewards.manageBulkRevoke') }}</button></template>
      <template #default="{row}"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><h3 class="break-words font-semibold">#{{ row.id }} · {{ row.template.name }}</h3><p class="mt-1 text-sm">{{ t('rewards.userID') }} #{{ row.user_id }} · {{ t(`rewards.${row.template.kind}`) }} · {{ t(`rewards.${row.status}`) }}</p><p class="mt-1 break-all text-sm">{{ templateModels(row.template).join(', ') || `${row.template.rate}×` }} · {{ row.remaining_tokens.toLocaleString() }} Token</p><p class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ t('rewards.manageSource') }}: {{ sourceLabel(row.source) }} · {{ t('rewards.ends') }}: {{ new Date(row.expires_at || row.claim_expires_at).toLocaleString() }}</p></div><button v-if="row.status!=='revoked'" class="btn btn-secondary min-h-11" :disabled="locked" @click="prepareRevoke(row.id)">{{ t('rewards.revoke') }}</button></div></template>
    </ManagementList>
    <ConfirmDialog :show="!!pending" :danger="pending==='revoke'" :title="t(pending==='grant'?'rewards.manageBulkGrant':pending==='discard'?'rewards.manageEndBatch':'rewards.manageBulkRevoke')" :message="confirmMessage" @cancel="cancel" @confirm="confirm"><label v-if="pending==='revoke'" class="block text-sm">{{ t('rewards.reason') }}<input v-model="revokeReason" class="input mt-1" :aria-label="t('rewards.reason')" maxlength="500"></label><p v-if="pending==='grant'" class="break-words text-sm">{{ grantJobs.map(v=>`#${v.id}`).join(', ') }}</p><p v-if="dialogError" role="alert" class="text-red-700 dark:text-red-300">{{ dialogError }}</p></ConfirmDialog>
  </section>
</template>
<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import {computed,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {rewardsAPI,type RewardCard,type CardTemplate} from '@/api/rewards'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ManagementList from './ManagementList.vue'
import BatchResults from './BatchResults.vue'
import TemplatePicker from './TemplatePicker.vue'
import RecipientPicker from './RecipientPicker.vue'
import {useManagementBatch} from './useManagementBatch'
import {readManagementSession,writeManagementSession} from './managementSession'
const props=defineProps<{cards:RewardCard[];templates:CardTemplate[];groups:{id:number;name:string}[];busy:boolean;reload:()=>Promise<void>}>()
const emit=defineEmits<{busy:[value:boolean]}>(),{t}=useI18n()
const {processing,results,refreshError,execute}=useManagementBatch(props.reload,v=>emit('busy',v))
const locked=computed(()=>props.busy||processing.value),selected=ref<(number|string)[]>([]),templateID=ref(0),recipientText=ref(''),grantReason=ref(''),revokeReason=ref(''),validation=ref(''),dialogError=ref(''),pending=ref(''),revokeIDs=ref<number[]>([])
type GrantJob={id:number;template:number;key:string;reason:string}
const savedJobs=readManagementSession<GrantJob[]>('grants',(v):v is GrantJob[]=>Array.isArray(v)&&v.length<=100&&v.every(j=>j&&Number.isSafeInteger(j.id)&&j.id>0&&Number.isSafeInteger(j.template)&&j.template>0&&typeof j.key==='string'&&j.key.length>=8&&typeof j.reason==='string'))
const grantJobs=ref<GrantJob[]>(savedJobs || [])
const recipientIDs=computed(()=>[...new Set(recipientText.value.trim().split(/[\s,，;；]+/).filter(v=>/^\d+$/.test(v)).map(Number).filter(v=>Number.isSafeInteger(v)&&v>0))])
const filters=computed(()=>[{key:'kind',label:t('rewards.kind'),options:['model_day','group_rate','model_tokens'].map(value=>({value,label:t(`rewards.${value}`)}))},{key:'status',label:t('rewards.manageStatus'),options:['unused','active','exhausted','expired','revoked'].map(value=>({value,label:t(`rewards.${value}`)}))},{key:'source',label:t('rewards.manageSource'),options:[...new Set(props.cards.map(v=>v.source))].map(value=>({value,label:sourceLabel(value)}))}])
function sourceLabel(v:string){return ['admin','code','farm','lottery'].includes(v)?t(`rewards.source_${v}`):v}
function cardSearch(v:RewardCard){return `${v.id} ${v.user_id} ${v.template.name} ${templateModels(v.template).join(', ')} ${v.template.group_ids.join(' ')} ${v.source}`}
function cardFilter(v:RewardCard,key:string){return key==='kind'?v.template.kind:key==='status'?v.status:v.source}
const confirmMessage=computed(()=>pending.value==='grant'?t('rewards.manageGrantConfirm',{count:grantJobs.value.length,name:props.templates.find(v=>v.id===grantJobs.value[0]?.template)?.name||''}):pending.value==='discard'?t('rewards.manageDiscardConfirm'):t('rewards.manageRevokeConfirm',{count:revokeIDs.value.length}))
let grantAttempted=!!savedJobs?.length
function prepareGrant(){
  if(locked.value)return
  validation.value='';dialogError.value='';grantAttempted=false
  const tokens=recipientText.value.trim().split(/[\s,，;；]+/),ids=tokens.map(Number)
  if(!tokens.length || tokens.some(v=>!/^\d+$/.test(v)) || ids.some(v=>!Number.isSafeInteger(v)||v<=0) || new Set(ids).size>100 || !grantReason.value.trim() || !props.templates.some(v=>v.id===templateID.value&&v.published)){validation.value=t('rewards.manageInvalidRecipients');return}
  grantJobs.value=[...new Set(ids)].map(id=>({id,template:templateID.value,key:crypto.randomUUID(),reason:grantReason.value.trim()}));pending.value='grant'
}
function cancel(){if(pending.value==='grant'&&!grantAttempted)grantJobs.value=[];pending.value='';dialogError.value=''}
function prepareRevoke(id?:number){dialogError.value='';revokeReason.value='';revokeIDs.value=id?[id]:props.cards.filter(v=>selected.value.includes(v.id)&&v.status!=='revoked').map(v=>v.id);pending.value='revoke'}
async function confirm(){
  if(locked.value)return
  const action=pending.value
  if(action==='discard'){try{writeManagementSession('grants');grantJobs.value=[];pending.value=''}catch{dialogError.value=t('rewards.manageSessionFailed')}return}
  if(action==='grant'){
    try{writeManagementSession('grants',grantJobs.value)}catch{dialogError.value=t('rewards.manageSessionFailed');return}
    grantAttempted=true;pending.value=''
    await execute([...grantJobs.value],v=>`#${v.id}`,v=>rewardsAPI.grant(v.id,v.template,v.key,v.reason),id=>{
      grantJobs.value=grantJobs.value.filter(v=>v.id!==id)
      // A stale receipt remains safe to replay if storage becomes unavailable.
      try{writeManagementSession('grants',grantJobs.value.length?grantJobs.value:undefined)}catch{validation.value=t('rewards.manageSessionFailed')}
    });return
  }
  if(!revokeReason.value.trim()){dialogError.value=t('rewards.manageReasonRequired');return}
  const reason=revokeReason.value.trim();pending.value=''
  await execute(revokeIDs.value.map(id=>({id})),v=>`#${v.id}`,v=>rewardsAPI.revoke(v.id,reason),id=>{selected.value=selected.value.filter(v=>v!==id)})
}
</script>
