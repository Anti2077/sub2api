<template>
  <div class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2">
      <label>{{ t('rewards.name') }}<input v-model="value.name" class="input mt-1" required maxlength="120"></label>
      <label>{{ t('rewards.kind') }}<select v-model="value.kind" class="input mt-1"><option v-for="k in kinds" :key="k" :value="k">{{ t('rewards.'+k) }}</option></select></label>
      <SearchMultiSelect :model-value="value.group_ids" :options="groups.map(g=>({value:g.id,label:g.name+' (#'+g.id+')'}))" :label="t('rewards.groups')" :placeholder="t('incentives.selectGroups')" :search-placeholder="t('incentives.searchGroups')" :empty-text="t('incentives.noGroups')" :remove-label="t('incentives.removeOption')" :disabled="disabled" @update:model-value="value.group_ids=$event.map(Number)" />
      <label v-if="value.kind==='group_rate'">{{ t('rewards.rate') }}<input v-model.number="value.rate" class="input mt-1" type="number" min="0" max="100" step="any" required></label>
    </div>
    <div v-if="value.kind!=='group_rate'" class="space-y-2">
      <SearchMultiSelect :model-value="templateModels(value)" :options="modelOptions" :label="t('rewards.model')" :placeholder="t('rewards.selectModels')" :search-placeholder="t('rewards.searchModels')" :empty-text="t('rewards.noModels')" :remove-label="t('incentives.removeOption')" :disabled="disabled" @update:model-value="setModels($event.map(String))" />
      <p v-if="loading" role="status" class="text-sm text-gray-500">{{ t('rewards.loading') }}</p>
      <p v-if="loadFailed" role="status" class="text-sm text-amber-700 dark:text-amber-300">{{ t('rewards.modelsLoadFailed') }}</p>
      <label class="block text-sm">{{ t('rewards.customModels') }}<textarea v-model="custom" class="input mt-1" rows="2" :placeholder="t('rewards.customModelsHint')" :disabled="disabled" /></label>
      <button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !custom.trim()" @click="addCustom">{{ t('rewards.addModels') }}</button>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t(value.kind==='model_tokens'?'rewards.sharedTokensNote':'rewards.modelsNote') }}</p>
      <p v-if="modelError" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ t('rewards.invalidModels') }}</p>
    </div>
    <div class="grid gap-4 sm:grid-cols-2">
      <label v-if="value.kind==='model_tokens'">{{ t('rewards.tokens') }}<input v-model.number="value.tokens" class="input mt-1" type="number" min="1" max="1000000000000" required></label>
      <div>
        <label class="block">{{ t('rewards.duration') }}<span class="mt-1 flex gap-2"><input v-model.number="duration" class="input min-w-0 flex-1" type="number" min="1" :max="unit==='days'?365:8760" step="1" required><select v-model="unit" class="input w-28" :aria-label="t('rewards.durationUnit')"><option value="days">{{ t('rewards.durationDays') }}</option><option v-if="value.kind==='model_tokens'" value="hours">{{ t('rewards.hours') }}</option></select></span></label>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('rewards.durationNote') }}</p>
      </div>
      <label>{{ t('rewards.claimDays') }}<input v-model.number="value.claim_days" class="input mt-1" type="number" min="1" max="365" required></label>
    </div>
    <label class="flex min-h-11 items-center gap-2"><input v-model="value.published" type="checkbox">{{ t('rewards.published') }}</label>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SearchMultiSelect from '@/components/common/SearchMultiSelect.vue'
import { adminAPI } from '@/api/admin'
import type { CardTemplate } from '@/api/rewards'
import { templateModels } from '@/components/rewards/templateValidation'
import { validModelIDs } from './templateValidation'
defineProps<{ groups:{id:number;name:string}[]; disabled:boolean }>()
const value=defineModel<CardTemplate>({required:true})
const {t}=useI18n(), kinds=['model_day','group_rate','model_tokens'] as const
const custom=ref(''), modelError=ref(false), candidates=ref<string[]>([]), loading=ref(false), loadFailed=ref(false)
const durationUnit=ref(value.value.hours%24===0?'days':'hours')
const unit=computed({get:()=>durationUnit.value,set:(v:string)=>{if(v==='days')value.value.hours=Math.max(24,Math.ceil(value.value.hours/24)*24);durationUnit.value=v}})
const duration=computed({get:()=>unit.value==='days'?value.value.hours/24:value.value.hours,set:(v:number)=>{value.value.hours=v*(unit.value==='days'?24:1)}})
watch(()=>value.value.kind,kind=>{if(kind!=='model_tokens'){unit.value='days';value.value.hours=Math.max(24,Math.ceil(value.value.hours/24)*24)}})
const modelOptions=computed(()=>[...new Set([...candidates.value,...templateModels(value.value)])].map(value=>({value,label:value})))
function setModels(models:string[]){value.value.models=[...new Set(models)];value.value.model=value.value.models[0]||'';modelError.value=false}
function addCustom(){const models=[...new Set([...templateModels(value.value),...custom.value.split(/[\s,，;；]+/u).filter(Boolean)])];if(!validModelIDs(models)){modelError.value=true;return false}setModels(models);custom.value='';return true}
function prepare(){if(value.value.kind==='group_rate')return true;if(custom.value.trim()&&!addCustom())return false;modelError.value=!validModelIDs(templateModels(value.value));return !modelError.value}
defineExpose({prepare})
watch(()=>[...value.value.group_ids],async(ids,_,onCleanup)=>{
  let current=true;onCleanup(()=>{current=false});candidates.value=[];loadFailed.value=false
  if(!ids.length){loading.value=false;return}
  loading.value=true
  const results=await Promise.allSettled(ids.map(id=>adminAPI.groups.getModelAllowlistCandidates(id)))
  if(!current)return
  candidates.value=[...new Set(results.flatMap(r=>r.status==='fulfilled'?r.value:[]))].filter(id=>validModelIDs([id]))
  loadFailed.value=results.some(r=>r.status==='rejected');loading.value=false
},{immediate:true})
</script>
