<template>
  <div class="space-y-4">
    <BatchResults :results="results" :refresh-error="refreshError" />
    <ManagementList v-model="selected" :items="items" :title="t('rewards.shop')" :busy="locked" :note="t('rewards.manageShopNote')" :search-text="v=>`${v.id} ${v.template.name} ${v.template.model} ${v.template.group_ids.join(' ')}`" :row-label="v=>`#${v.id} · ${v.template.name}`" :filters="filters" :filter-value="filterValue" :selectable="()=>true">
      <template #header><button class="btn btn-primary min-h-11" :disabled="locked" @click="emit('edit')">{{ t('rewards.newShop') }}</button></template>
      <template #actions="{selected:chosen}"><button class="btn btn-secondary min-h-11" :disabled="locked || !chosen.some(v=>!v.published)" @click="pending=true">{{ t('rewards.bulkPublish') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked || !chosen.some(v=>v.published)" @click="pending=false">{{ t('rewards.bulkUnpublish') }}</button></template>
      <template #default="{row}"><div class="flex flex-wrap items-center justify-between gap-3"><div class="min-w-0 space-y-1"><h3 class="break-words font-semibold">#{{ row.id }} · {{ row.template.name }}</h3><p class="text-sm">{{ t(`rewards.${row.template.kind}`) }} · {{ t(row.published?'rewards.published':'rewards.draft') }} · v{{ row.version }}</p><p class="text-sm">{{ row.price }} {{ t('rewards.crystals') }} · {{ t('rewards.stock') }} {{ row.stock }} · {{ t('rewards.daily') }} {{ row.daily_limit }} / {{ t('rewards.total') }} {{ row.total_limit }}</p><p class="break-all text-xs text-gray-600 dark:text-gray-300">{{ row.template.model || `${row.template.rate}×` }} · {{ row.starts_at ? new Date(row.starts_at).toLocaleString() : '—' }} → {{ row.ends_at ? new Date(row.ends_at).toLocaleString() : '—' }}</p></div><button class="btn btn-secondary min-h-11" :disabled="locked" @click="emit('edit',row)">{{ t('rewards.edit') }}</button></div></template>
    </ManagementList>
    <ConfirmDialog :show="pending!==undefined" :title="t(pending?'rewards.bulkPublish':'rewards.bulkUnpublish')" :message="t('rewards.manageShopConfirm',{count:changes.length})" @cancel="pending=undefined" @confirm="save" />
  </div>
</template>
<script setup lang="ts">
import {computed,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {rewardsAPI,type Shop} from '@/api/rewards'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ManagementList from './ManagementList.vue'
import BatchResults from './BatchResults.vue'
import {useManagementBatch} from './useManagementBatch'
const props=defineProps<{items:Shop[];busy:boolean;reload:()=>Promise<void>}>(),emit=defineEmits<{edit:[value?:Shop];busy:[value:boolean]}>(),{t}=useI18n()
const {processing,results,refreshError,execute}=useManagementBatch(props.reload,v=>emit('busy',v))
const locked=computed(()=>props.busy||processing.value),selected=ref<(number|string)[]>([]),pending=ref<boolean>()
const changes=computed(()=>props.items.filter(v=>selected.value.includes(v.id)&&v.published!==pending.value))
const filters=computed(()=>[{key:'kind',label:t('rewards.kind'),options:['model_day','group_rate','model_tokens'].map(value=>({value,label:t(`rewards.${value}`)}))},{key:'status',label:t('rewards.templateStatus'),options:[{value:'published',label:t('rewards.published')},{value:'draft',label:t('rewards.draft')}]},{key:'stock',label:t('rewards.stock'),options:[{value:'in',label:t('rewards.manageInStock')},{value:'out',label:t('rewards.manageOutStock')}]}])
function filterValue(v:Shop,k:string){return k==='kind'?v.template.kind:k==='status'?(v.published?'published':'draft'):(v.stock>0?'in':'out')}
async function save(){if(locked.value||pending.value===undefined)return;const target=pending.value,rows=changes.value.map(v=>({...v}));pending.value=undefined;await execute(rows,v=>`#${v.id} · ${v.template.name}`,v=>rewardsAPI.shopStatus(v.id,v.version,target),id=>{selected.value=selected.value.filter(v=>v!==id)})}
</script>
