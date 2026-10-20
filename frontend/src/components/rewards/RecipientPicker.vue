<template>
  <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-600" @toggle="onToggle">
    <summary class="min-h-11 cursor-pointer py-2 font-medium">{{ title || t('rewards.manageChooseRecipients') }}</summary>
    <div class="mt-3 space-y-3" :aria-busy="loading">
      <div class="grid items-end gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto]">
        <label class="min-w-0 text-sm">{{ t('rewards.manageRecipientSearch') }}<input v-model="query" type="search" class="input mt-1" :disabled="disabled || loading" :aria-label="t('rewards.manageRecipientSearch')" @keydown.enter.prevent="search"></label>
        <label class="min-w-0 text-sm">{{ t('rewards.manageUserStatus') }}<select v-model="status" class="input mt-1" :disabled="disabled || loading" @change="search"><option value="">{{ t('rewards.all') }}</option><option value="active">{{ t('rewards.manageActiveUsers') }}</option><option value="disabled">{{ t('rewards.manageDisabledUsers') }}</option></select></label>
        <button type="button" class="btn btn-secondary min-h-11 w-full sm:w-auto" :disabled="disabled || loading" @click="search">{{ t('rewards.manageFindUsers') }}</button>
      </div>
      <p v-if="error" role="alert" class="break-words text-sm text-red-700 dark:text-red-300">{{ error }}</p>
      <p role="status" class="text-sm">{{ t('rewards.manageSelectedRecipients',{count:modelValue.length}) }}</p>
      <div class="divide-y divide-gray-200 rounded-lg border border-gray-200 dark:divide-dark-600 dark:border-dark-600">
        <label class="flex min-h-11 items-center gap-3 p-3 text-sm"><input type="checkbox" :checked="allPageSelected" :indeterminate="somePageSelected" :disabled="disabled || loading || !rows.length" @change="togglePage">{{ t('rewards.selectPage') }}</label>
        <label v-for="user in rows" :key="user.id" class="flex min-h-11 items-start gap-3 p-3 text-sm"><input class="mt-1" type="checkbox" :checked="modelValue.includes(user.id)" :disabled="disabled || loading" :aria-label="t('rewards.manageSelect',{name:`#${user.id} · ${user.username || user.email}`})" @change="toggle(user.id)"><span class="min-w-0 break-all"><strong>#{{ user.id }} · {{ user.username || user.email }}</strong><span v-if="user.username" class="block text-gray-600 dark:text-gray-300">{{ user.email }}</span><span class="block text-gray-600 dark:text-gray-300">{{ t(user.status==='active'?'rewards.manageActiveUsers':'rewards.manageDisabledUsers') }}</span></span></label>
        <p v-if="!rows.length && !loading" class="p-4 text-sm">{{ t('rewards.manageRecipientEmpty') }}</p>
      </div>
      <footer class="flex flex-wrap items-center justify-between gap-2 text-sm"><span>{{ t('rewards.templatePage',{count:total,page,pages}) }}</span><div class="flex gap-2"><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || loading || page<=1" @click="load(page-1)">{{ t('rewards.previousPage') }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || loading || page>=pages" @click="load(page+1)">{{ t('rewards.nextPage') }}</button></div></footer>
    </div>
  </details>
</template>
<script setup lang="ts">
import {computed,onBeforeUnmount,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {adminAPI} from '@/api/admin'
import {rewardError} from '@/api/rewards'
import type {AdminUser} from '@/types'
const props=withDefaults(defineProps<{modelValue:number[];disabled:boolean;title?:string;maxSelected?:number}>(),{title:'',maxSelected:100})
const emit=defineEmits<{'update:modelValue':[value:number[]]}>(),{t}=useI18n()
const query=ref(''),status=ref<''|'active'|'disabled'>('active'),page=ref(1),total=ref(0),rows=ref<AdminUser[]>([]),loading=ref(false),error=ref('')
const pages=computed(()=>Math.max(1,Math.ceil(total.value/12)))
const allPageSelected=computed(()=>rows.value.length>0&&rows.value.every(v=>props.modelValue.includes(v.id)))
const somePageSelected=computed(()=>!allPageSelected.value&&rows.value.some(v=>props.modelValue.includes(v.id)))
let initialized=false,controller:AbortController|undefined
async function load(next=1){
  if(props.disabled||loading.value)return
  controller=new AbortController();loading.value=true;error.value=''
  try{
    const term=query.value.trim()
    if(/^\d+$/.test(term)&&Number.isSafeInteger(Number(term))){
      try{const user=await adminAPI.users.getById(Number(term));rows.value=!status.value||user.status===status.value?[user]:[];total.value=rows.value.length;page.value=1}
      catch(e){if((e as {response?:{status:number}}).response?.status!==404)throw e;rows.value=[];total.value=0;page.value=1}
    }else{const result=await adminAPI.users.list(next,12,{search:term,status:status.value||undefined},{signal:controller.signal});rows.value=result.items;total.value=result.total;page.value=next}
    initialized=true
  }
  catch(e){if(!controller.signal.aborted)error.value=rewardError(e)}finally{loading.value=false}
}
function search(){void load(1)}
function onToggle(event:Event){if((event.target as HTMLDetailsElement).open&&!initialized&&!props.disabled){initialized=true;void load()}}
function update(ids:number[]){if(props.disabled||loading.value)return;if(ids.length>props.maxSelected){error.value=t('rewards.manageRecipientLimit',{count:props.maxSelected});return}error.value='';emit('update:modelValue',ids)}
function toggle(id:number){update(props.modelValue.includes(id)?props.modelValue.filter(v=>v!==id):[...props.modelValue,id])}
function togglePage(){const ids=rows.value.map(v=>v.id);update(allPageSelected.value?props.modelValue.filter(id=>!ids.includes(id)):[...new Set([...props.modelValue,...ids])])}
onBeforeUnmount(()=>controller?.abort())
</script>
