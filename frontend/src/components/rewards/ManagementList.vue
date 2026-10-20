<template>
  <section class="space-y-4" :aria-label="title" :aria-busy="busy">
    <header class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-lg font-semibold">{{ title }} · {{ items.length }}</h2><slot name="header" /></header>
    <p v-if="note" class="text-sm text-gray-600 dark:text-gray-300">{{ note }}</p>
    <div class="card grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-3">
      <label class="text-sm">{{ t('rewards.manageSearch') }}<input v-model="query" class="input mt-1" type="search" :aria-label="t('rewards.manageSearch')" :placeholder="searchHint" :disabled="busy"></label>
      <label v-for="filter in filters" :key="filter.key" class="text-sm">{{ filter.label }}<select v-model="values[filter.key]" class="input mt-1" :aria-label="filter.label" :disabled="busy"><option value="">{{ t('rewards.all') }}</option><option v-for="option in filter.options" :key="option.value" :value="option.value">{{ option.label }}</option></select></label>
    </div>
    <div v-if="selectable" class="flex flex-wrap items-center gap-2">
      <span role="status" class="mr-auto text-sm">{{ t('rewards.manageSelected', { count: modelValue.length }) }}</span><slot name="actions" :selected="selectedRows" />
      <button type="button" class="btn btn-secondary min-h-11" :disabled="busy || !modelValue.length" @click="emit('update:modelValue', [])">{{ t('rewards.clearSelection') }}</button>
    </div>
    <div v-if="selectable" class="flex flex-wrap items-center gap-2 text-sm">
      <span class="mr-auto text-gray-600 dark:text-gray-300">{{ t('rewards.manageSelectionScope', { selected: selectedRows.length, matching: filtered.length }) }}</span>
      <button type="button" class="btn btn-secondary min-h-11" :disabled="busy || !matchingEligible.length || allMatchingSelected" @click="selectMatching">{{ t('rewards.manageSelectMatching', { count: matchingEligible.length }) }}</button>
      <button type="button" class="btn btn-secondary min-h-11" :disabled="busy || !eligible.length" @click="invertPage">{{ t('rewards.manageInvertPage') }}</button>
    </div>
    <div v-if="hasFilters" class="flex justify-end"><button type="button" class="btn btn-secondary min-h-11" :disabled="busy" @click="resetFilters">{{ t('rewards.manageResetFilters') }}</button></div>
    <div class="card overflow-hidden">
      <label v-if="selectable" class="flex min-h-12 items-center gap-3 border-b border-gray-200 px-4 dark:border-dark-600"><input type="checkbox" :checked="allSelected" :indeterminate="someSelected" :disabled="busy || !eligible.length" @change="togglePage"><span class="text-sm">{{ t('rewards.selectPage') }}</span></label>
      <p v-if="!filtered.length" class="p-8 text-center text-sm">{{ t('rewards.manageEmpty') }}</p>
      <article v-for="row in pageRows" :key="row.id" class="flex min-w-0 items-start gap-3 border-b border-gray-200 p-4 last:border-b-0 dark:border-dark-600" :data-row-id="row.id">
        <label v-if="selectable" class="flex min-h-11 min-w-6 items-center"><input type="checkbox" :checked="modelValue.includes(row.id)" :disabled="busy || !selectable(row)" :aria-label="t('rewards.manageSelect', { name: rowLabel(row) })" @change="toggle(row.id)"></label>
        <div class="min-w-0 flex-1"><slot :row="row" /></div>
      </article>
    </div>
    <footer class="flex flex-wrap items-center justify-between gap-3 text-sm"><span>{{ t('rewards.templatePage', { count: filtered.length, page, pages }) }}</span><label class="flex items-center gap-2">{{ t('rewards.managePageSize') }}<select v-model.number="pageSize" class="input w-20" :disabled="busy" :aria-label="t('rewards.managePageSize')"><option v-for="size in [12,24,48]" :key="size" :value="size">{{ size }}</option></select></label><div class="flex gap-2"><button type="button" class="btn btn-secondary min-h-11" :disabled="busy || page===1" @click="page--">{{ t('rewards.previousPage') }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="busy || page===pages" @click="page++">{{ t('rewards.nextPage') }}</button></div></footer>
  </section>
</template>
<script setup lang="ts" generic="T extends { id: number | string }">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
export interface ManagementFilter { key: string; label: string; options: {value: string; label: string}[] }
const props = withDefaults(defineProps<{items:T[];title:string;busy?:boolean;note?:string;searchHint?:string;searchText:(row:T)=>string;rowLabel:(row:T)=>string;filters?:ManagementFilter[];filterValue?:(row:T,key:string)=>string;selectable?:(row:T)=>boolean;modelValue?:(number|string)[]}>(), {busy:false,note:'',searchHint:'',filters:()=>[],modelValue:()=>[]})
const emit=defineEmits<{'update:modelValue':[value:(number|string)[]]}>(),{t}=useI18n()
const query=ref(''),page=ref(1),pageSize=ref(12),values=reactive<Record<string,string>>({})
watch(()=>props.filters,filters=>{for(const f of filters){if(values[f.key]===undefined)values[f.key]=''}},{immediate:true})
const filtered=computed(()=>props.items.filter(row=>(!query.value.trim()||props.searchText(row).toLowerCase().includes(query.value.trim().toLowerCase()))&&props.filters.every(f=>!values[f.key]||props.filterValue?.(row,f.key)===values[f.key])))
const pages=computed(()=>Math.max(1,Math.ceil(filtered.value.length/pageSize.value))),pageRows=computed(()=>filtered.value.slice((page.value-1)*pageSize.value,page.value*pageSize.value))
const matchingEligible=computed(()=>filtered.value.filter(row=>props.selectable?.(row)))
const allMatchingSelected=computed(()=>!!matchingEligible.value.length&&matchingEligible.value.every(row=>props.modelValue.includes(row.id)))
const hasFilters=computed(()=>!!query.value||Object.values(values).some(Boolean))
const eligible=computed(()=>pageRows.value.filter(row=>props.selectable?.(row))),allSelected=computed(()=>!!eligible.value.length&&eligible.value.every(row=>props.modelValue.includes(row.id))),someSelected=computed(()=>!allSelected.value&&eligible.value.some(row=>props.modelValue.includes(row.id)))
const selectedRows=computed(()=>props.items.filter(row=>props.modelValue.includes(row.id)&&props.selectable?.(row)))
watch([query,values],()=>{page.value=1;emit('update:modelValue',[])},{deep:true})
watch(pageSize,()=>{page.value=1})
watch(pages,n=>{page.value=Math.min(page.value,n)})
watch(()=>props.items,rows=>{const ids=props.modelValue.filter(id=>rows.some(row=>row.id===id&&props.selectable?.(row)));if(ids.length!==props.modelValue.length)emit('update:modelValue',ids)})
function resetFilters(){if(props.busy)return;query.value='';for(const key of Object.keys(values))values[key]=''}
function selectMatching(){if(props.busy)return;emit('update:modelValue',[...new Set([...props.modelValue,...matchingEligible.value.map(row=>row.id)])])}
function invertPage(){if(props.busy)return;const ids=eligible.value.map(row=>row.id);emit('update:modelValue',[...props.modelValue.filter(id=>!ids.includes(id)),...ids.filter(id=>!props.modelValue.includes(id))])}
function toggle(id:number|string){if(props.busy)return;emit('update:modelValue',props.modelValue.includes(id)?props.modelValue.filter(v=>v!==id):[...props.modelValue,id])}
function togglePage(){if(props.busy)return;const ids=eligible.value.map(row=>row.id);emit('update:modelValue',allSelected.value?props.modelValue.filter(id=>!ids.includes(id)):[...new Set([...props.modelValue,...ids])])}
async function reveal(id:number|string){resetFilters();await nextTick();const index=filtered.value.findIndex(v=>v.id===id);if(index>=0)page.value=Math.floor(index/pageSize.value)+1;await nextTick()}
defineExpose({reveal})
</script>
