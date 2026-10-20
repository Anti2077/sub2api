<template>
  <div class="space-y-3">
  <div v-if="invalid.length" role="alert" class="rounded-lg border border-red-200 p-3 text-sm text-red-700 dark:border-red-800 dark:text-red-300"><p>{{ t('rewards.manageInvalidCrops',{count:invalid.length}) }}</p><div class="mt-2 flex flex-wrap gap-2"><button v-for="row in invalid" :key="row.id" type="button" class="btn btn-secondary min-h-11" @click="list?.reveal(row.id)">{{ row.name || row.id }}</button></div></div>
  <ManagementList ref="list" v-model="selected" :items="modelValue" :title="t('rewards.crops')" :busy="disabled" :note="t('rewards.manageCropNote')" :search-text="v=>`${v.id} ${v.name} ${v.hours}`" :row-label="v=>v.name" :filters="[{key:'status',label:t('rewards.manageStatus'),options:[{value:'enabled',label:t('rewards.manageEnabled')},{value:'disabled',label:t('rewards.manageDisabled')}]}]" :filter-value="v=>v.enabled?'enabled':'disabled'" :selectable="()=>true">
    <template #actions><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !modelValue.some(v=>selected.includes(v.id)&&!v.enabled)" @click="setEnabled(true)">{{ t('rewards.manageEnable') }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || !modelValue.some(v=>selected.includes(v.id)&&v.enabled)" @click="setEnabled(false)">{{ t('rewards.manageDisable') }}</button></template>
    <template #default="{row}"><div class="grid gap-3 sm:grid-cols-4"><label>{{ t('rewards.name') }}<input v-model="row.name" class="input mt-1" :disabled="disabled" required maxlength="120"></label><label>{{ t('rewards.hours') }}<input v-model.number="row.hours" class="input mt-1" :disabled="disabled" type="number" min="1" max="720" required></label><label>{{ t('rewards.yield') }}<input v-model.number="row.yield" class="input mt-1" :disabled="disabled" type="number" min="1" max="1000000" required></label><label class="flex min-h-11 items-center gap-2"><input v-model="row.enabled" type="checkbox" :disabled="disabled">{{ t('rewards.manageEnabled') }}</label></div><p class="mt-2 text-xs text-gray-600 dark:text-gray-300">{{ row.id }}</p></template>
  </ManagementList>
  </div>
</template>
<script setup lang="ts">
import {computed,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import type {Crop} from '@/api/rewards'
import ManagementList from './ManagementList.vue'
import {invalidCrops} from './managementValidation'
const props=defineProps<{modelValue:Crop[];disabled:boolean}>(),emit=defineEmits<{'update:modelValue':[value:Crop[]]}>(),{t}=useI18n(),selected=ref<(number|string)[]>([])
const list=ref<{reveal:(id:string)=>Promise<void>}>(),invalid=computed(()=>invalidCrops(props.modelValue))
function setEnabled(enabled:boolean){emit('update:modelValue',props.modelValue.map(v=>selected.value.includes(v.id)?{...v,enabled}:v));selected.value=[]}
</script>
