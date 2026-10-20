<template>
  <ManagementList :items="items" :title="title" :note="t('rewards.manageRecentRecords')" :search-text="v=>JSON.stringify(v)" :row-label="v=>`#${v.id}`" :filters="filters" :filter-value="(v,k)=>String(v[k]??'')">
    <template #default="{row}"><div class="space-y-2 text-sm"><div class="flex flex-wrap justify-between gap-2"><strong class="break-all">#{{ row.id }} · {{ kindLabel(row.kind) }}</strong><span>{{ new Date(String(row.created_at)).toLocaleString() }}</span></div><p>{{ t('rewards.userID') }} #{{ row.user_id || '—' }}<span v-if="row.actor_id"> · {{ t('rewards.manageActor') }} #{{ row.actor_id }}</span><span v-if="row.card_id"> · {{ t('rewards.manageCardID') }} #{{ row.card_id }}</span><strong v-if="row.delta!==undefined" class="ml-2">{{ row.delta>0?'+':'' }}{{ row.delta.toLocaleString() }}</strong><span v-if="row.spend!==undefined"> · ${{ Number(row.spend).toFixed(8) }}</span></p><p v-if="row.reference" class="break-all text-xs">{{ row.reference }}</p><details v-if="row.detail"><summary class="cursor-pointer">{{ t('rewards.manageDetails') }}</summary><pre class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-all text-xs">{{ JSON.stringify(row.detail,null,2) }}</pre></details></div></template>
  </ManagementList>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import ManagementList from './ManagementList.vue'
export interface ManagedRecord {id:number;kind:string;created_at:string;user_id?:number;actor_id?:number;card_id?:number;delta?:number;reference?:string;detail?:unknown;[key:string]:unknown}
const props=defineProps<{items:ManagedRecord[];title:string}>(),{t,te}=useI18n()
function kindLabel(kind:string){const key=`rewards.manageEvents.${kind}`;return te(key)?t(key):kind}
const filters=computed(()=>[{key:'kind',label:t('rewards.kind'),options:[...new Set(props.items.map(v=>v.kind))].map(value=>({value,label:kindLabel(value)}))}])
</script>
