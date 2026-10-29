<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div><h2 class="text-lg font-semibold">{{ t('rewards.templates') }} · {{ templates.length }}</h2><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('rewards.templateManageNote') }}</p></div>
      <button class="btn btn-primary min-h-11" :disabled="locked" @click="emit('edit')">{{ t('rewards.newTemplate') }}</button>
    </div>
    <div class="card grid gap-3 p-4 sm:grid-cols-3">
      <label class="text-sm">{{ t('rewards.searchTemplates') }}<input v-model="search" class="input mt-1" type="search" :aria-label="t('rewards.searchTemplates')" :placeholder="t('rewards.templateSearchHint')" :disabled="locked"></label>
      <label class="text-sm">{{ t('rewards.kind') }}<select v-model="kind" class="input mt-1" :aria-label="t('rewards.kind')" :disabled="locked"><option value="all">{{ t('rewards.all') }}</option><option v-for="k in kinds" :key="k" :value="k">{{ t(`rewards.${k}`) }}</option></select></label>
      <label class="text-sm">{{ t('rewards.templateStatus') }}<select v-model="status" class="input mt-1" :aria-label="t('rewards.templateStatus')" :disabled="locked"><option value="all">{{ t('rewards.all') }}</option><option value="published">{{ t('rewards.published') }}</option><option value="draft">{{ t('rewards.draft') }}</option></select></label>
    </div>
    <div class="flex flex-wrap items-center gap-2 text-sm">
      <span role="status" class="mr-auto">{{ t('rewards.selectedTemplates', { count: selected.length }) }}</span>
      <button class="btn btn-secondary min-h-11" :disabled="locked || !templates.some(v=>selected.includes(v.id)&&!v.published)" @click="pending=true">{{ t('rewards.bulkPublish') }}</button>
      <button class="btn btn-secondary min-h-11" :disabled="locked || !templates.some(v=>selected.includes(v.id)&&v.published)" @click="pending=false">{{ t('rewards.bulkUnpublish') }}</button>
      <button class="btn btn-secondary min-h-11" :disabled="locked || !selected.length" @click="selected=[]">{{ t('rewards.clearSelection') }}</button>
    </div>
    <div class="flex flex-wrap items-center gap-2 text-sm"><span class="mr-auto text-gray-600 dark:text-gray-300">{{ t('rewards.manageSelectionScope',{selected:selected.length,matching:filtered.length}) }}</span><button type="button" class="btn btn-secondary min-h-11" :disabled="locked || !filtered.length || filtered.every(v=>selected.includes(v.id))" @click="selected=[...new Set([...selected,...filtered.map(v=>v.id)])]">{{ t('rewards.manageSelectMatching',{count:filtered.length}) }}</button><button type="button" class="btn btn-secondary min-h-11" :disabled="locked || !pageItems.length" @click="invertPage">{{ t('rewards.manageInvertPage') }}</button><button v-if="search || kind!=='all' || status!=='all'" type="button" class="btn btn-secondary min-h-11" :disabled="locked" @click="search='';kind='all';status='all'">{{ t('rewards.manageResetFilters') }}</button></div>
    <p v-if="result" role="status" class="text-sm">{{ result }}</p>
    <p v-if="failure" role="alert" class="break-words text-sm text-red-700 dark:text-red-300">{{ failure }} {{ t('rewards.bulkTemplateRetry') }}</p>
    <details v-if="outcomes.length" class="card p-4 text-sm" open>
      <summary class="cursor-pointer">{{ t('rewards.bulkTemplateDetails') }}</summary>
      <ul class="mt-3 space-y-2">
        <li v-for="item in outcomes" :key="item.id" class="break-words" :class="item.failed ? 'text-red-700 dark:text-red-300' : 'text-emerald-800 dark:text-emerald-200'">#{{ item.id }} · {{ item.name }} — {{ t(item.failed ? 'rewards.bulkTemplateFailed' : 'rewards.bulkTemplateSaved') }}</li>
      </ul>
    </details>
    <div class="card overflow-hidden">
      <label class="flex min-h-12 items-center gap-3 border-b border-gray-200 px-4 dark:border-dark-600"><input type="checkbox" :checked="allSelected" :indeterminate="someSelected" :disabled="locked || !pageItems.length" @change="togglePage"><span class="text-sm">{{ t('rewards.selectPage') }}</span></label>
      <p v-if="!filtered.length" class="p-10 text-center text-sm">{{ t('rewards.noTemplates') }}</p>
      <article v-for="v in pageItems" :key="v.id" class="flex items-start gap-3 border-b border-gray-200 p-4 last:border-b-0 dark:border-dark-600" :data-template-id="v.id">
        <label class="flex min-h-11 min-w-6 items-center"><input v-model="selected" type="checkbox" :value="v.id" :disabled="locked" :aria-label="t('rewards.selectTemplate', { name: v.name, id: v.id })"></label>
        <div class="grid min-w-0 flex-1 gap-3 md:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_auto] md:items-center">
          <div class="min-w-0"><h3 class="break-words font-semibold">{{ v.name }}</h3><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ v.id }} · v{{ v.version }} · {{ t(`rewards.${v.kind}`) }}</p><span class="mt-2 inline-flex rounded-full px-2 py-1 text-xs" :class="v.published ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'">{{ t(v.published?'rewards.published':'rewards.draft') }}</span></div>
          <div class="min-w-0 text-sm"><p class="break-all">{{ v.kind==='group_rate' ? `${v.rate}× · ${v.hours}h` : templateModels(v).join(', ') }}</p><p class="mt-1 text-gray-500 dark:text-gray-400">{{ v.kind==='model_tokens' ? `${v.tokens.toLocaleString()} Token · ${v.hours}h` : v.kind==='model_day' ? t('rewards.freeText')+` · ${v.hours}h` : '' }}</p><p class="mt-1 break-words text-xs text-gray-500 dark:text-gray-400">{{ groupNames(v) }}</p></div>
          <div class="flex flex-wrap gap-2"><button class="btn btn-secondary min-h-11" :disabled="locked" @click="emit('edit', v)">{{ t('rewards.edit') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked" @click="duplicate(v)">{{ t('rewards.duplicateTemplate') }}</button></div>
        </div>
      </article>
    </div>
    <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
      <span>{{ t('rewards.templatePage', { count: filtered.length, page, pages }) }}</span><label class="flex items-center gap-2">{{ t('rewards.managePageSize') }}<select v-model.number="pageSize" class="input w-20" :disabled="locked" :aria-label="t('rewards.managePageSize')"><option v-for="size in [12,24,48]" :key="size" :value="size">{{ size }}</option></select></label><div class="flex gap-2"><button class="btn btn-secondary min-h-11" :disabled="locked || page===1" @click="page--">{{ t('rewards.previousPage') }}</button><button class="btn btn-secondary min-h-11" :disabled="locked || page===pages" @click="page++">{{ t('rewards.nextPage') }}</button></div>
    </div>
    <ConfirmDialog :show="pending!==undefined" :title="t(pending?'rewards.bulkPublish':'rewards.bulkUnpublish')" :message="t('rewards.bulkTemplateConfirm', { count: changes.length, action: t(pending?'rewards.published':'rewards.draft') })" @cancel="pending=undefined" @confirm="applyBatch" />
  </div>
</template>
<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { rewardsAPI, rewardError, type CardTemplate } from '@/api/rewards'
const props = defineProps<{ templates: CardTemplate[]; groups: { id: number; name: string }[]; busy: boolean; reload: () => Promise<void> }>()
const emit = defineEmits<{ edit: [value?: CardTemplate]; busy: [value: boolean] }>()
const { t } = useI18n()
const kinds = ['model_day', 'group_rate', 'model_tokens'] as const
const search = ref(''), kind = ref('all'), status = ref('all'), page = ref(1), selected = ref<number[]>([])
const pageSize=ref(12)
const processing = ref(false), pending = ref<boolean>(), result = ref(''), failure = ref('')
const outcomes = ref<{ id: number; name: string; failed: boolean }[]>([])
const locked = computed(() => props.busy || processing.value)
const groupNames = (v: CardTemplate) => v.group_ids.map(id => props.groups.find(g => g.id===id)?.name || `#${id}`).join(', ')
const filtered = computed(() => props.templates.filter(v => {
  const q = search.value.trim().toLowerCase()
  return (kind.value==='all' || v.kind===kind.value) && (status.value==='all' || v.published===(status.value==='published')) && (!q || `${v.id} ${v.name} ${templateModels(v).join(', ')} ${groupNames(v)}`.toLowerCase().includes(q))
}))
const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length/pageSize.value)))
const pageItems = computed(() => filtered.value.slice((page.value-1)*pageSize.value, page.value*pageSize.value))
const allSelected = computed(() => pageItems.value.length>0 && pageItems.value.every(v => selected.value.includes(v.id)))
const someSelected = computed(() => !allSelected.value && pageItems.value.some(v => selected.value.includes(v.id)))
const changes = computed(() => props.templates.filter(v => selected.value.includes(v.id) && v.published!==pending.value))
watch([search,kind,status], () => { page.value=1; selected.value=[]; result.value=''; failure.value=''; outcomes.value=[] })
watch(pages, n => { page.value=Math.min(page.value,n) })
watch(pageSize,()=>{page.value=1})
watch(() => props.templates, rows => { selected.value=selected.value.filter(id => rows.some(v => v.id===id)) })
function togglePage() {
  const ids = pageItems.value.map(v => v.id)
  selected.value=allSelected.value ? selected.value.filter(id => !ids.includes(id)) : [...new Set([...selected.value,...ids])]
}
function invertPage(){if(locked.value)return;const ids=pageItems.value.map(v=>v.id);selected.value=[...selected.value.filter(id=>!ids.includes(id)),...ids.filter(id=>!selected.value.includes(id))]}
function duplicate(v: CardTemplate) {
  emit('edit', { ...v, id: 0, version: 0, name: t('rewards.templateCopy', { name: v.name }).slice(0,120), group_ids: [...v.group_ids], models: [...templateModels(v)], published: false })
}
async function applyBatch() {
  if (locked.value || pending.value===undefined) return
  const target=pending.value, rows=changes.value.map(v => ({...v, group_ids:[...v.group_ids], published:target}))
  pending.value=undefined; processing.value=true; emit('busy',true); result.value=''; failure.value=''; outcomes.value=[]
  selected.value=rows.map(v => v.id)
  let success=0
  const failed: string[]=[]
  try {
    for (const row of rows) {
      try { await rewardsAPI.saveTemplate(row); success++; selected.value=selected.value.filter(id => id!==row.id); outcomes.value.push({id:row.id,name:row.name,failed:false}) }
      catch(e) { failed.push(`#${row.id}: ${rewardError(e)}`); outcomes.value.push({id:row.id,name:row.name,failed:true}) }
    }
    result.value=t('rewards.bulkTemplateResult', { success, failed: failed.length })
    failure.value=failed.join(' · ')
    try { await props.reload() } catch(e) { failure.value=[failure.value,t('rewards.manageRefreshFailed'),rewardError(e)].filter(Boolean).join(' · ') }
  } finally { processing.value=false; emit('busy',false) }
}
</script>
