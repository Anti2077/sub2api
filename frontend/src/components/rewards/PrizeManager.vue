<template>
  <div class="space-y-4">
    <div v-if="invalid.length" role="alert" class="rounded-lg border border-red-200 p-3 text-sm text-red-700 dark:border-red-800 dark:text-red-300"><p>{{ t('rewards.manageInvalidPrizes',{count:invalid.length}) }}</p><div class="mt-2 flex flex-wrap gap-2"><button v-for="row in invalid" :key="row.id" type="button" class="btn btn-secondary min-h-11" @click="list?.reveal(row.id)">{{ row.name || row.id }}</button></div></div>
    <ManagementList ref="list" v-model="selected" :items="modelValue" :title="t('incentives.prizes')" :busy="disabled" :note="t('rewards.managePrizeNote')" :search-text="searchText" :row-label="v=>v.name || v.id" :filters="filters" :filter-value="(v,k)=>k==='type'?(v.card_template_id?'card':'balance'):(v.enabled?'enabled':'disabled')" :selectable="()=>true">
      <template #header><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || modelValue.length>=8" @click="add">{{ t('incentives.addPrize') }}</button></template>
      <template #actions><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !modelValue.some(v=>selected.includes(v.id)&&!v.enabled)" @click="setEnabled(true)">{{ t('rewards.manageEnable') }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !modelValue.some(v=>selected.includes(v.id)&&v.enabled)" @click="setEnabled(false)">{{ t('rewards.manageDisable') }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !selected.length || modelValue.length-selected.length<2" @click="confirmRemove=true">{{ t('rewards.managePrizeRemove') }}</button></template>
      <template #default="{row}"><div class="space-y-3"><div class="grid gap-3 sm:grid-cols-2"><label>{{ t('incentives.prizeName') }}<input v-model="row.name" class="input mt-1" :disabled="disabled" required maxlength="40"></label><label>{{ t('rewards.managePrizeType') }}<select :value="row.card_template_id?'card':'balance'" class="input mt-1" :disabled="disabled" @change="changeType(row,($event.target as HTMLSelectElement).value)"><option value="balance">{{ t('rewards.balancePrize') }}</option><option value="card">{{ t('rewards.cardPrize') }}</option></select></label><label v-if="!row.card_template_id">{{ t('incentives.amount') }}<input v-model.number="row.reward_amount" class="input mt-1" :disabled="disabled" type="number" min="0" max="1000000" step="0.00000001" required></label><label>{{ t('incentives.weight') }}<input v-model.number="row.weight" class="input mt-1" :disabled="disabled" type="number" min="1" max="1000000000" required></label></div><TemplatePicker v-if="row.card_template_id" :model-value="row.card_template_id>0?row.card_template_id:0" :templates="templates" :groups="groups" :disabled="disabled" @update:model-value="row.card_template_id=$event" /><div class="flex flex-wrap items-center justify-between gap-3 text-sm"><label class="flex min-h-11 items-center gap-2"><input v-model="row.enabled" type="checkbox" :disabled="disabled">{{ t('incentives.enabled') }}</label><span>{{ t('rewards.manageProbability') }}: {{ probability(row) }}%</span></div></div></template>
    </ManagementList>
    <p v-if="!modelValue.some(v=>v.enabled)" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ t('rewards.manageMinPrizes') }}</p>
    <ConfirmDialog :show="confirmRemove" :title="t('rewards.managePrizeRemove')" :message="t('rewards.managePrizeRemoveConfirm',{count:selected.length})" @cancel="confirmRemove=false" @confirm="remove" />
  </div>
</template>
<script setup lang="ts">
import {computed,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import type {DailyLotteryPrize} from '@/api/dailyLottery'
import type {CardTemplate} from '@/api/rewards'
import ManagementList from './ManagementList.vue'
import {invalidPrizes} from './managementValidation'
import TemplatePicker from './TemplatePicker.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
const props=defineProps<{modelValue:DailyLotteryPrize[];templates:CardTemplate[];groups?:{id:number;name:string}[];disabled:boolean}>(),emit=defineEmits<{'update:modelValue':[value:DailyLotteryPrize[]]}>(),{t}=useI18n(),selected=ref<(number|string)[]>([]),confirmRemove=ref(false)
const list=ref<{reveal:(id:string)=>Promise<void>}>(),invalid=computed(()=>invalidPrizes(props.modelValue,props.templates))
const filters=computed(()=>[{key:'type',label:t('rewards.managePrizeType'),options:[{value:'balance',label:t('rewards.balancePrize')},{value:'card',label:t('rewards.cardPrize')}]},{key:'status',label:t('rewards.manageStatus'),options:[{value:'enabled',label:t('rewards.manageEnabled')},{value:'disabled',label:t('rewards.manageDisabled')}]}])
function searchText(v:DailyLotteryPrize){const c=props.templates.find(c=>c.id===v.card_template_id);return `${v.id} ${v.name} ${c?.name||''} ${c?.model||''}`}
function add(){emit('update:modelValue',[...props.modelValue,{id:crypto.randomUUID(),name:'',reward_amount:0,weight:1,enabled:true}])}
function setEnabled(enabled:boolean){emit('update:modelValue',props.modelValue.map(v=>selected.value.includes(v.id)?{...v,enabled}:v));selected.value=[]}
function remove(){if(props.modelValue.length-selected.value.length<2)return;emit('update:modelValue',props.modelValue.filter(v=>!selected.value.includes(v.id)));selected.value=[];confirmRemove.value=false}
function changeType(v:DailyLotteryPrize,type:string){v.card_template_id=type==='card'?-1:undefined;v.reward_amount=0}
function probability(v:DailyLotteryPrize){const total=props.modelValue.filter(v=>v.enabled).reduce((a,v)=>a+Math.max(0,v.weight),0);return(v.enabled&&total>0?Math.max(0,v.weight)/total*100:0).toFixed(2)}
</script>
