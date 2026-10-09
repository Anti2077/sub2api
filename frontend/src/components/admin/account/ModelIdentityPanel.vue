<template>
  <BaseDialog :show="show" :title="`${t('admin.modelIdentity.title')} · ${account?.name ?? ''}`" width="wide" @close="emit('close')">
    <div class="space-y-5">
      <p v-if="error" role="alert" class="break-words text-sm text-red-600">{{ error }}</p>
      <form class="grid grid-cols-1 gap-3 border-b border-gray-200 pb-5 dark:border-dark-600 sm:grid-cols-2" @submit.prevent="saveConfiguration">
        <label class="text-sm">{{ t('admin.modelIdentity.testUser') }}
          <input v-model="search" type="search" class="input mt-1 w-full" :placeholder="t('admin.modelIdentity.searchUser')" @input="searchUsers" />
          <select v-model="userID" required class="input mt-2 w-full"><option :value="0" disabled>{{ t('common.select') }}</option><option v-for="u in users" :key="u.id" :value="u.id">{{ u.username || u.email }} (#{{ u.id }})</option></select>
        </label>
        <label class="text-sm">{{ t('admin.modelIdentity.billingGroup') }}
          <select v-model="groupID" required class="input mt-1 w-full"><option :value="0" disabled>{{ t('common.select') }}</option><option v-for="g in compatibleGroups" :key="g.id" :value="g.id">{{ g.name }}</option></select>
        </label>
        <div class="sm:col-span-2"><button type="submit" class="btn btn-primary" :disabled="busy || !userID || !groupID">{{ t('common.save') }}</button></div>
        <p v-if="configuration" class="break-words text-xs text-gray-500 sm:col-span-2">{{ configuration.key_name }} (#{{ configuration.api_key_id }})</p>
      </form>
      <form class="grid grid-cols-1 gap-3 border-b border-gray-200 pb-5 dark:border-dark-600 sm:grid-cols-2" @submit.prevent="savePlan">
        <label class="text-sm">{{ t('admin.modelIdentity.requestModel') }}<input v-model="draft.request_model" required maxlength="256" class="input mt-1 w-full" /></label>
        <label class="text-sm">{{ t('admin.modelIdentity.expectedModel') }}<select v-model="draft.expected_model" required class="input mt-1 w-full"><option value="" disabled>{{ t('common.select') }}</option><option v-for="m in catalog" :key="m.id" :value="m.id">{{ m.name }}</option></select></label>
        <label class="text-sm">{{ t('admin.modelIdentity.interval') }}<input v-model.number="draft.interval_minutes" type="number" min="15" max="10080" required class="input mt-1 w-full" /></label>
        <label class="flex items-center gap-2 text-sm"><input v-model="draft.enabled" type="checkbox" />{{ t('admin.modelIdentity.scheduled') }}</label>
        <div class="flex flex-wrap gap-2 sm:col-span-2"><button type="submit" class="btn btn-primary" :disabled="busy || !configuration || !catalog.length">{{ editingID ? t('common.save') : t('admin.scheduledTests.addPlan') }}</button><button v-if="editingID" type="button" class="btn btn-secondary" @click="resetDraft">{{ t('common.cancel') }}</button></div>
      </form>
      <div v-for="p in plans" :key="p.id" class="border-b border-gray-200 pb-4 dark:border-dark-600">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="min-w-0 text-sm"><div class="break-words font-medium">{{ p.request_model }} → {{ p.expected_model }}</div><div class="mt-1 text-xs text-gray-500">{{ p.enabled ? t('admin.modelIdentity.scheduled') : t('admin.modelIdentity.paused') }} · {{ p.interval_minutes }} {{ t('admin.modelIdentity.minutes') }}</div></div>
          <div class="flex gap-1"><button :title="t('admin.modelIdentity.run')" :aria-label="t('admin.modelIdentity.run')" class="btn btn-secondary p-2" :disabled="busy" @click="runPlan(p)"><Icon name="play" size="sm" /></button><button :title="t('common.edit')" :aria-label="t('common.edit')" class="btn btn-secondary p-2" @click="editPlan(p)"><Icon name="edit" size="sm" /></button><button :title="t('common.delete')" :aria-label="t('common.delete')" class="btn btn-secondary p-2" :disabled="busy" @click="deletePlan(p)"><Icon name="trash" size="sm" /></button><button :title="t('admin.modelIdentity.history')" :aria-label="t('admin.modelIdentity.history')" class="btn btn-secondary p-2" @click="loadHistory(p.id)"><Icon name="clock" size="sm" /></button></div>
        </div>
        <div class="mt-2 grid gap-1 text-xs text-gray-500 sm:grid-cols-2"><span>{{ t('admin.scheduledTests.lastRun') }}: {{ time(p.last_run_at) }}</span><span>{{ t('admin.scheduledTests.nextRun') }}: {{ time(p.next_run_at) }}</span></div>
      </div>
      <div v-if="runs.length" class="space-y-2">
        <h4 class="text-sm font-medium">{{ t('admin.modelIdentity.history') }}</h4>
        <div v-for="r in runs" :key="r.id" class="border-b border-gray-200 py-2 dark:border-dark-600">
          <div class="flex flex-wrap items-center justify-between gap-2 text-sm"><span>{{ time(r.created_at) }} · {{ t(`admin.modelIdentity.${String(r.status === 'completed' ? r.report?.verdict || r.status : r.status)}`) }} <span v-if="r.status === 'running'">({{ r.probe_count ?? 0 }})</span></span><button v-if="['queued','running','cancelling'].includes(r.status)" class="btn btn-secondary" @click="cancelRun(r)">{{ t('common.cancel') }}</button><button class="btn btn-secondary" @click="loadReport(r)">{{ t('admin.modelIdentity.report') }}</button></div>
        </div>
      </div>
      <div v-if="selectedReport" class="min-w-0 space-y-3">
        <h4 class="text-sm font-medium">{{ t('admin.modelIdentity.report') }} #{{ selectedReport.id }}</h4>
        <dl class="grid grid-cols-1 gap-2 text-sm sm:grid-cols-2">
          <div><dt class="text-gray-500">{{ t('admin.modelIdentity.expectedModel') }}</dt><dd class="break-words">{{ selectedReport.expected_model }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.modelIdentity.detectedModel') }}</dt><dd class="break-words">{{ selectedReport.report?.detected_model || '-' }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.modelIdentity.engineVersion') }}</dt><dd class="break-words">{{ selectedReport.report?.engine_version || '-' }} ({{ selectedReport.report?.engine_commit || '-' }})</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.modelIdentity.baselineVersion') }}</dt><dd class="break-words">{{ selectedReport.report?.baseline_version || '-' }}</dd></div>
        </dl>
        <p v-if="selectedReport.report?.error" role="alert" class="break-words text-sm text-red-600">{{ selectedReport.report.error }}</p>
        <details v-for="(probe,index) in selectedReport.probes" :key="index" class="border-b border-gray-200 py-2 text-sm dark:border-dark-600">
          <summary class="cursor-pointer break-words">{{ probe.probe_id || index + 1 }} · {{ probe.status }}</summary>
          <dl class="mt-2 space-y-1 text-xs">
            <div>{{ t('admin.modelIdentity.targetAccount') }}: {{ probe.target_account_id }}</div>
            <div class="break-words">{{ t('admin.modelIdentity.requestModel') }}: {{ probe.request_model }}</div>
            <div class="break-words">{{ t('admin.modelIdentity.upstreamModel') }}: {{ probe.evidence?.upstream_model || '-' }}</div>
            <div class="break-words">{{ t('admin.modelIdentity.requestID') }}: {{ probe.evidence?.request_id || '-' }}</div>
            <div>{{ t('admin.modelIdentity.tokenUsage') }}: {{ JSON.stringify(probe.response?.usage || {}) }}</div>
          </dl>
          <p v-if="probe.error" class="mt-2 break-words text-red-600">{{ probe.error }}</p>
          <pre class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs">{{ [probe.prompt, JSON.stringify(probe.response, null, 2)].join('\n\n') }}</pre>
        </details>
        <details><summary class="cursor-pointer text-sm">{{ t('admin.modelIdentity.evidence') }}</summary><pre class="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-700">{{ JSON.stringify(selectedReport.report, null, 2) }}</pre></details>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { Icon } from '@/components/icons'
import type { Account, AdminUser, Group } from '@/types'
import * as api from '@/api/admin/modelIdentity'
import * as usersAPI from '@/api/admin/users'

const props = defineProps<{show:boolean;account:Account|null;groups:Group[]}>()
const emit = defineEmits<{(e:'close'):void}>()
const { t } = useI18n()
const userID = ref(0), groupID = ref(0), search = ref(''), error = ref(''), busy = ref(false), editingID = ref(0)
const users = ref<AdminUser[]>([]), catalog = ref<api.IdentityModel[]>([]), plans = ref<api.IdentityPlan[]>([]), runs = ref<api.IdentityRun[]>([])
const configuration = ref<api.IdentityConfig|null>(null), selectedReport = ref<api.IdentityRun|null>(null), historyPlanID = ref(0)
const draft = reactive({request_model:'',expected_model:'',interval_minutes:120,enabled:false})
const compatibleGroups = computed(() => props.groups.filter(g => props.account?.group_ids?.includes(g.id) && g.platform === props.account?.platform && !g.claude_code_only))
let searchTimer:ReturnType<typeof setTimeout>|undefined
let poll:ReturnType<typeof setInterval>|undefined
let searchVersion = 0
let viewVersion = 0
let historyVersion = 0
function time(value?:string) { return value ? new Date(value).toLocaleString() : '-' }
function message(e:unknown) { return typeof e === 'object' && e !== null && 'message' in e ? String(e.message) : String(e) }
async function action(fn:()=>Promise<void>) { busy.value=true;error.value='';try {await fn()} catch(e) {error.value=message(e)} finally {busy.value=false} }
async function fetchUsers() { const version=++searchVersion; const view=viewVersion;const result=await usersAPI.list(1,20,{search:search.value,status:'active'});if(version===searchVersion&&view===viewVersion)users.value=result.items }
function searchUsers() { clearTimeout(searchTimer);searchTimer=setTimeout(()=>{void fetchUsers().catch(e=>{error.value=message(e)})},300) }
async function refreshPlans() { if(props.account)plans.value=await api.plans(props.account.id) }
async function saveConfiguration() { await action(async()=>{if(!props.account)return;configuration.value=await api.saveConfig(props.account.id,{user_id:userID.value,group_id:groupID.value});await refreshPlans()}) }
function resetDraft() { editingID.value=0;Object.assign(draft,{request_model:'',expected_model:'',interval_minutes:120,enabled:false}) }
function editPlan(p:api.IdentityPlan) { editingID.value=p.id;Object.assign(draft,p) }
async function savePlan() { await action(async()=>{if(!props.account)return;const body={...draft,account_id:props.account.id};if(editingID.value)await api.updatePlan(editingID.value,body);else await api.savePlan(body);resetDraft();await refreshPlans()}) }
async function deletePlan(p:api.IdentityPlan) { await action(async()=>{await api.deletePlan(p.id);await refreshPlans()}) }
async function loadHistory(id:number) { const version=++historyVersion;const view=viewVersion;historyPlanID.value=id;await action(async()=>{const result=await api.history(id);if(version===historyVersion&&view===viewVersion)runs.value=result}) }
async function runPlan(p:api.IdentityPlan) { await action(async()=>{await api.run(p.id);historyPlanID.value=p.id;runs.value=await api.history(p.id)}) }
async function cancelRun(r:api.IdentityRun) { await action(async()=>{await api.cancel(r.id);runs.value=await api.history(r.plan_id)}) }
async function loadReport(r:api.IdentityRun) { await action(async()=>{selectedReport.value=await api.runStatus(r.id)}) }
watch(()=>[props.show,props.account?.id],async()=>{
  const version=++viewVersion;historyVersion++;searchVersion++;clearInterval(poll);clearTimeout(searchTimer);selectedReport.value=null;runs.value=[];plans.value=[];configuration.value=null;catalog.value=[];users.value=[];userID.value=0;groupID.value=0;search.value='';historyPlanID.value=0;resetDraft();error.value=''
  if(!props.show||!props.account)return
  const accountID=props.account.id
  await action(async()=>{
    await fetchUsers();if(version!==viewVersion)return
    const loadedPlans=await api.plans(accountID);if(version!==viewVersion)return;plans.value=loadedPlans
    try{const config=await api.config(accountID);if(version!==viewVersion)return;configuration.value=config;userID.value=config.user_id;groupID.value=config.group_id;const u=await usersAPI.getById(config.user_id);if(version!==viewVersion)return;if(!users.value.some(x=>x.id===u.id))users.value.push(u)}catch(e){const failure=e as {status?:number;response?:{status:number}};if((failure.status??failure.response?.status)!==404)throw e}
    const result=await api.models();if(version!==viewVersion)return;catalog.value=result.models
    if(plans.value.length){historyPlanID.value=plans.value[0].id;const history=await api.history(historyPlanID.value);if(version===viewVersion)runs.value=history}
  })
  if(version!==viewVersion)return
  poll=setInterval(()=>{const plan=historyPlanID.value;if(props.show&&plan)void Promise.all([api.history(plan),api.plans(accountID)]).then(([history,latestPlans])=>{if(version===viewVersion&&plan===historyPlanID.value){runs.value=history;plans.value=latestPlans}}).catch(e=>{if(version===viewVersion)error.value=message(e)})},3000)
},{immediate:true})
onUnmounted(()=>{clearInterval(poll);clearTimeout(searchTimer);searchVersion++;viewVersion++;historyVersion++})
</script>
