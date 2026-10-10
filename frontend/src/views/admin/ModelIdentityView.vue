<template>
  <AppLayout>
    <div class="space-y-6 pb-12">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.modelIdentity.overviewTitle') }}</h1>
          <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.overviewDescription') }}</p>
        </div>
        <button class="btn btn-secondary min-h-11" type="button" :disabled="loading" @click="loadRows()"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button>
      </header>
      <p v-if="loadError" role="alert" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">{{ loadError }}</p>
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <div v-for="stat in stats" :key="stat.label" class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ stat.label }}</p>
          <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ stat.value }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.currentPage') }}</p>
        </div>
      </div>
      <section class="card overflow-hidden" :aria-busy="loading">
        <div class="flex flex-col gap-3 border-b border-gray-200 p-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
          <div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.modelIdentity.accountOverview') }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.accountOverviewHint') }}</p></div>
          <div class="flex flex-wrap gap-2">
            <label class="sr-only" for="identity-search">{{ t('admin.modelIdentity.filterAccounts') }}</label>
            <input id="identity-search" v-model="query" class="input min-h-11 w-full lg:w-60" type="search" :placeholder="t('admin.modelIdentity.filterAccounts')" />
            <label class="sr-only" for="identity-filter">{{ t('admin.modelIdentity.filterStatus') }}</label>
            <select id="identity-filter" v-model="statusFilter" class="input min-h-11 w-full lg:w-40">
              <option value="all">{{ t('admin.modelIdentity.allStatuses') }}</option>
              <option v-for="status in filterStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
            </select>
          </div>
        </div>
        <p v-if="loading && !rows.length" role="status" class="p-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</p>
        <p v-else-if="!filteredRows.length" class="p-8 text-center text-sm text-gray-500">{{ t('common.noData') }}</p>
        <div v-else class="hidden overflow-x-auto lg:block">
          <table class="w-full min-w-[1150px] table-fixed text-left text-sm">
            <colgroup><col class="w-[130px]" /><col class="w-[150px]" /><col class="w-[160px]" /><col class="w-[280px]" /><col class="w-[140px]" /><col class="w-[100px]" /><col class="w-[130px]" /></colgroup>
            <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-900/50 dark:text-gray-400"><tr>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.account') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.expectedModel') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.lastResult') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.recentRuns') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.nextRun') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.modelIdentity.keyStatus') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('common.actions') }}</th>
            </tr></thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <template v-for="row in filteredRows" :key="row.account.id">
                <tr :data-account-id="row.account.id" class="align-top hover:bg-gray-50/80 dark:hover:bg-dark-800/60">
                  <td class="px-4 py-4"><strong class="block text-gray-900 dark:text-white">{{ row.account.name }}</strong><span class="mt-1 block text-xs text-gray-500">#{{ row.account.id }} · {{ groupName(row) }}</span><p v-if="row.error" role="alert" class="mt-2 max-w-64 text-xs text-red-600 dark:text-red-400">{{ row.error }}</p></td>
                  <td class="px-4 py-4"><code class="block break-all text-xs">{{ row.latest?.expected_model || row.plans[0]?.expected_model || '—' }}</code><span class="mt-1 block text-xs text-gray-500">{{ row.latest?.request_model || row.plans[0]?.request_model || '—' }}</span><span v-if="row.plans.length > 1" class="mt-1 block text-xs text-gray-500">{{ t('admin.modelIdentity.planCount', { count: row.plans.length }) }}</span></td>
                  <td class="px-4 py-4"><span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="identityStatusClass(row.status)">{{ statusLabel(row.status) }}</span><span class="mt-2 block break-all text-xs">{{ row.latest?.report?.detected_model || '—' }}</span><time class="mt-1 block whitespace-nowrap text-xs text-gray-500">{{ formatTime(row.latest?.created_at) }}</time></td>
                  <td class="px-4 py-4"><IdentityRecentRuns :runs="row.history.slice(0, 5)" @report="openReport" /></td>
                  <td class="px-4 py-4 text-xs"><time>{{ formatTime(nextRun(row)) }}</time><span class="mt-1 block text-gray-500">{{ t('admin.modelIdentity.enabledPlanCount', { count: row.plans.filter(plan => plan.enabled).length }) }}</span></td>
                  <td class="max-w-48 px-4 py-4 text-xs"><span :class="row.config?.configuration_error ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-300'">{{ row.config ? (row.config.configuration_error ? t('admin.modelIdentity.keyError') : t('admin.modelIdentity.keyReady')) : t('admin.modelIdentity.notConfigured') }}</span><span v-if="row.config" class="mt-1 block break-words" :title="row.config.key_name">#{{ row.config.api_key_id }}</span></td>
                  <td class="px-4 py-4"><div class="flex flex-col gap-2"><button class="btn btn-secondary min-h-11 text-xs" type="button" @click="configure(row)">{{ t('admin.modelIdentity.manageAccount') }}</button><button class="btn btn-ghost min-h-11 text-xs" type="button" :aria-expanded="expandedRows.has(row.account.id)" :aria-controls="`identity-history-${row.account.id}`" @click="toggleHistory(row.account.id)">{{ expandedRows.has(row.account.id) ? t('common.collapse') : t('admin.modelIdentity.history') }}</button></div></td>
                </tr>
                <tr v-if="expandedRows.has(row.account.id)" :id="`identity-history-${row.account.id}`"><td colspan="7" class="bg-gray-50/70 px-4 py-4 dark:bg-dark-900/30"><div class="space-y-3">
                  <div class="flex flex-wrap gap-2"><button v-for="plan in row.plans" :key="plan.id" class="btn btn-secondary min-h-11 text-xs" :disabled="Boolean(actionBusy) || isActive(row.latest)" type="button" @click="runPlan(plan)"><Icon name="play" size="sm" />{{ plan.request_model }} · {{ t('admin.modelIdentity.run') }}</button></div>
                  <IdentityHistoryTable :runs="row.history" :busy="Boolean(actionBusy)" @report="openReport" @cancel="cancelRun" />
                </div></td></tr>
              </template>
            </tbody>
          </table>
        </div>
        <div class="divide-y divide-gray-200 dark:divide-dark-700 lg:hidden">
          <article v-for="row in filteredRows" :key="row.account.id" class="space-y-4 p-4" :data-account-id="row.account.id">
            <div class="flex items-start justify-between gap-3"><div><h3 class="font-semibold text-gray-900 dark:text-white">{{ row.account.name }}</h3><p class="mt-1 text-xs text-gray-500">#{{ row.account.id }} · {{ groupName(row) }}</p></div><span class="shrink-0 rounded-full px-2 py-1 text-xs" :class="identityStatusClass(row.status)">{{ statusLabel(row.status) }}</span></div>
            <p v-if="row.error" role="alert" class="text-sm text-red-600">{{ row.error }}</p>
            <dl class="grid grid-cols-2 gap-3 text-xs"><div><dt class="text-gray-500">{{ t('admin.modelIdentity.expectedModel') }}</dt><dd class="mt-1 break-all">{{ row.latest?.expected_model || row.plans[0]?.expected_model || '—' }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.detectedModel') }}</dt><dd class="mt-1 break-all">{{ row.latest?.report?.detected_model || '—' }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.nextRun') }}</dt><dd class="mt-1">{{ formatTime(nextRun(row)) }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.keyStatus') }}</dt><dd class="mt-1">{{ row.config ? (row.config.configuration_error ? t('admin.modelIdentity.keyError') : `#${row.config.api_key_id}`) : t('admin.modelIdentity.notConfigured') }}</dd></div></dl>
            <div><h4 class="mb-2 text-xs text-gray-500">{{ t('admin.modelIdentity.recentRuns') }}</h4><IdentityRecentRuns :runs="row.history.slice(0, 5)" @report="openReport" /></div>
            <div class="flex flex-wrap gap-2"><button class="btn btn-secondary min-h-11" type="button" @click="configure(row)">{{ t('admin.modelIdentity.manageAccount') }}</button><button v-for="plan in row.plans" :key="plan.id" class="btn btn-secondary min-h-11 text-xs" :disabled="Boolean(actionBusy) || isActive(row.latest)" type="button" @click="runPlan(plan)">{{ plan.request_model }} · {{ t('admin.modelIdentity.run') }}</button><button v-if="isActive(row.latest)" class="btn btn-secondary min-h-11" :disabled="Boolean(actionBusy)" type="button" @click="cancelRun(row.latest!)">{{ t('common.cancel') }}</button></div>
          </article>
        </div>
        <Pagination v-if="total > pageSize" :page="page" :page-size="pageSize" :total="total" :show-page-size-selector="false" @update:page="changePage" />
      </section>
      <section class="grid gap-6 lg:grid-cols-3">
        <div class="card p-5 lg:col-span-2"><h2 class="font-semibold">{{ t('admin.modelIdentity.recentActivity') }}</h2><p class="mt-1 text-xs text-gray-500">{{ t('admin.modelIdentity.currentPage') }}</p><p v-if="!events.length" class="mt-5 text-sm text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p><button v-for="event in events" :key="event.run.id" type="button" class="mt-3 flex min-h-11 w-full flex-wrap items-center justify-between gap-2 rounded-lg border border-gray-100 p-3 text-left dark:border-dark-700" @click="openReport(event.run)"><span class="text-sm">{{ event.name }} <span class="ml-2 rounded-full px-2 py-1 text-xs" :class="identityStatusClass(identityRunStatus(event.run))">{{ statusLabel(identityRunStatus(event.run)) }}</span><span class="mt-1 block break-all text-xs text-gray-500">{{ event.run.request_model }} → {{ event.run.report?.detected_model || '—' }}</span></span><time class="text-xs text-gray-500">{{ formatTime(event.run.created_at) }}</time></button></div>
        <div class="card p-5"><h2 class="font-semibold">{{ t('admin.modelIdentity.scheduleSummary') }}</h2><dl class="mt-5 space-y-4 text-sm"><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('admin.modelIdentity.enabledPlans') }}</dt><dd>{{ enabledPlans.length }}</dd></div><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('admin.modelIdentity.runningNow') }}</dt><dd>{{ rows.filter(row => isActive(row.latest)).length }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.nextWindow') }}</dt><dd class="mt-1">{{ formatTime(scheduleNext) }}</dd></div></dl><p class="mt-5 text-xs text-gray-500">{{ t('admin.modelIdentity.scheduleHint') }}</p></div>
      </section>
      <ModelIdentityPanel :show="Boolean(selectedAccount)" :account="selectedAccount" :groups="groups" @close="closeConfiguration" @changed="loadRows()" />
      <BaseDialog :show="reportOpen" :title="t('admin.modelIdentity.report')" width="wide" @close="closeReport">
        <p v-if="reportLoading" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
        <p v-if="reportError" role="alert" class="text-sm text-red-600">{{ reportError }}</p>
        <div v-if="selectedReport" class="space-y-4">
          <dl class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2"><div><dt class="text-gray-500">{{ t('admin.modelIdentity.lastResult') }}</dt><dd class="mt-1">{{ statusLabel(identityRunStatus(selectedReport)) }} · {{ formatTime(selectedReport.created_at) }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.expectedModel') }}</dt><dd class="mt-1 break-all">{{ selectedReport.expected_model }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.detectedModel') }}</dt><dd class="mt-1 break-all">{{ selectedReport.report?.detected_model || '—' }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.engineVersion') }}</dt><dd class="mt-1">{{ selectedReport.report?.engine_version || '—' }}</dd></div></dl>
          <p v-if="selectedReport.report?.error" role="alert" class="text-sm text-red-600">{{ selectedReport.report.error }}</p>
          <p v-if="reportReason" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-900">{{ reportReason }}</p>
          <button v-if="isActive(selectedReport)" class="btn btn-secondary" type="button" :disabled="Boolean(actionBusy)" @click="cancelRun(selectedReport)">{{ t('common.cancel') }}</button>
          <details v-for="(probe, index) in selectedReport.probes" :key="index" class="border-b border-gray-200 py-2 dark:border-dark-700"><summary class="min-h-11 cursor-pointer break-words text-sm">{{ probe.probe_id || index + 1 }} · {{ probe.status }}</summary><dl class="space-y-1 text-xs"><div>{{ t('admin.modelIdentity.targetAccount') }}: {{ probe.target_account_id }}</div><div>{{ t('admin.modelIdentity.requestModel') }}: {{ probe.request_model }}</div><div>{{ t('admin.modelIdentity.upstreamModel') }}: {{ probe.evidence?.upstream_model || '—' }}</div><div>{{ t('admin.modelIdentity.requestID') }}: {{ probe.evidence?.request_id || '—' }}</div></dl><pre class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs">{{ JSON.stringify(probe.response, null, 2) }}</pre></details>
          <details><summary class="min-h-11 cursor-pointer text-sm">{{ t('admin.modelIdentity.evidence') }}</summary><pre class="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-900">{{ JSON.stringify(selectedReport.report, null, 2) }}</pre></details>
        </div>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import ModelIdentityPanel from '@/components/admin/account/ModelIdentityPanel.vue'
import IdentityRecentRuns from '@/components/admin/account/IdentityRecentRuns.vue'
import IdentityHistoryTable from '@/components/admin/account/IdentityHistoryTable.vue'
import { adminAPI } from '@/api/admin'
import * as identityAPI from '@/api/admin/modelIdentity'
import { identityRunStatus, identityStatusClass, isActive, type IdentityStatus } from '@/utils/modelIdentity'
import type { Account, AccountListItem, Group } from '@/types'

type IdentityRow = { account: AccountListItem; config: identityAPI.IdentityConfig | null; plans: identityAPI.IdentityPlan[]; history: identityAPI.IdentityRun[]; latest?: identityAPI.IdentityRun; status: IdentityStatus; error: string }
const { t, locale } = useI18n()
const rows = ref<IdentityRow[]>([]), groups = ref<Group[]>([])
const loading = ref(false), loadError = ref(''), query = ref(''), statusFilter = ref('all')
const page = ref(1), total = ref(0), pageSize = 20
const selectedAccount = ref<Account | null>(null), expandedRows = ref(new Set<number>()), actionBusy = ref(0)
const reportOpen = ref(false), reportLoading = ref(false), reportError = ref(''), selectedReport = ref<identityAPI.IdentityRun | null>(null)
let loadVersion = 0, reportVersion = 0, queryTimer: ReturnType<typeof setTimeout> | undefined, poll: ReturnType<typeof setInterval> | undefined
const filterStatuses: IdentityStatus[] = ['matched', 'mismatched', 'inconclusive', 'running', 'queued', 'configuration_error', 'request_error', 'service_error', 'notConfigured', 'untested']
const filteredRows = computed(() => rows.value.filter(row => statusFilter.value === 'all' || row.status === statusFilter.value))
const enabledPlans = computed(() => rows.value.flatMap(row => row.plans).filter(plan => plan.enabled))
const scheduleNext = computed(() => enabledPlans.value.map(plan => plan.next_run_at).filter((value): value is string => Boolean(value)).sort()[0])
const stats = computed(() => [
  { label: t('admin.modelIdentity.totalAccounts'), value: rows.value.length },
  { label: t('admin.modelIdentity.matchedAccounts'), value: rows.value.filter(row => row.status === 'matched').length },
  { label: t('admin.modelIdentity.attentionNeeded'), value: rows.value.filter(row => ['mismatched', 'inconclusive', 'configuration_error', 'request_error', 'service_error', 'timed_out'].includes(row.status)).length },
  { label: t('admin.modelIdentity.enabledPlans'), value: enabledPlans.value.length },
])
const events = computed(() => rows.value.flatMap(row => row.history.map(run => ({ name: row.account.name, run }))).sort((a, b) => +new Date(b.run.created_at) - +new Date(a.run.created_at)).slice(0, 5))
const reportReason = computed(() => {
  const assessment = selectedReport.value?.report?.identityAssessment as { verdict?: { reasoning?: string } } | undefined
  return assessment?.verdict?.reasoning || ''
})
function statusLabel(status: IdentityStatus) { return t(`admin.modelIdentity.${status}`) }
function message(error: unknown) { return error && typeof error === 'object' && 'message' in error ? String(error.message) : String(error) }
function formatTime(value?: string) { return value ? new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US', { hour12: false }) : '—' }
function groupName(row: IdentityRow) { return row.config ? groups.value.find(group => group.id === row.config?.group_id)?.name || `#${row.config.group_id}` : t('admin.modelIdentity.notConfigured') }
function nextRun(row: IdentityRow) { return row.plans.filter(plan => plan.enabled).map(plan => plan.next_run_at).filter((value): value is string => Boolean(value)).sort()[0] }
function toggleHistory(id: number) { const next = new Set(expandedRows.value); next.has(id) ? next.delete(id) : next.add(id); expandedRows.value = next }
async function loadRow(account: AccountListItem): Promise<IdentityRow> {
  const row: IdentityRow = { account, config: null, plans: [], history: [], status: 'notConfigured', error: '' }
  try {
    try { row.config = await identityAPI.config(account.id) } catch (error) { const failure = error as { status?: number; response?: { status?: number } }; if ((failure.status ?? failure.response?.status) !== 404) throw error }
    if (!row.config) return row
    row.plans = await identityAPI.plans(account.id)
    const histories: identityAPI.IdentityRun[] = []
    for (const plan of row.plans) histories.push(...await identityAPI.history(plan.id))
    row.history = histories.sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at)).slice(0, 50)
    row.latest = row.history[0]
    row.status = row.config.configuration_error ? 'configuration_error' : row.latest ? identityRunStatus(row.latest) : 'untested'
    row.error = row.config.configuration_error || ''
  } catch (error) { row.status = 'service_error'; row.error = message(error) }
  return row
}
async function loadRows() {
  const version = ++loadVersion
  loading.value = true; loadError.value = ''
  try {
    const result = await adminAPI.accounts.list(page.value, pageSize, { lite: 'true', search: query.value, sort_by: 'id', sort_order: 'desc' })
    const loaded: IdentityRow[] = []
    let cursor = 0
    await Promise.all(Array.from({ length: Math.min(4, result.items.length) }, async () => {
      while (cursor < result.items.length) { const account = result.items[cursor++]; const row = await loadRow(account); if (version !== loadVersion) return; loaded.push(row) }
    }))
    if (version !== loadVersion) return
    const positions = new Map(result.items.map((account, index) => [account.id, index]))
    rows.value = loaded.sort((a, b) => (positions.get(a.account.id) || 0) - (positions.get(b.account.id) || 0)); total.value = result.total
  } catch (error) { if (version === loadVersion) { rows.value = []; loadError.value = message(error) } }
  finally { if (version === loadVersion) loading.value = false }
}
function changePage(value: number) { page.value = value; void loadRows() }
async function configure(row: IdentityRow) {
  try { selectedAccount.value = await adminAPI.accounts.getById(row.account.id) }
  catch (error) { loadError.value = message(error) }
}
function closeConfiguration() { selectedAccount.value = null; void loadRows() }
async function runPlan(plan: identityAPI.IdentityPlan) {
  actionBusy.value = plan.id
  try { await identityAPI.run(plan.id); await loadRows() }
  catch (error) { loadError.value = message(error) }
  finally { actionBusy.value = 0 }
}
async function cancelRun(run: identityAPI.IdentityRun) {
  actionBusy.value = run.id
  try { await identityAPI.cancel(run.id); await loadRows(); if (selectedReport.value?.id === run.id) selectedReport.value = await identityAPI.runStatus(run.id) }
  catch (error) { loadError.value = message(error) }
  finally { actionBusy.value = 0 }
}
async function openReport(run: identityAPI.IdentityRun) {
  const version = ++reportVersion
  reportOpen.value = true; reportLoading.value = true; reportError.value = ''; selectedReport.value = null
  try { const result = await identityAPI.runStatus(run.id); if (version === reportVersion) selectedReport.value = result }
  catch (error) { if (version === reportVersion) reportError.value = message(error) }
  finally { if (version === reportVersion) reportLoading.value = false }
}
function closeReport() { reportVersion++; reportOpen.value = false; selectedReport.value = null }
watch(query, () => { clearTimeout(queryTimer); queryTimer = setTimeout(() => { page.value = 1; void loadRows() }, 300) })
onMounted(async () => {
  await Promise.allSettled([loadRows(), adminAPI.groups.getAll().then(result => { groups.value = result }).catch(error => { loadError.value = message(error) })])
  poll = setInterval(() => { if (!document.hidden && !loading.value && !selectedAccount.value && rows.value.some(row => isActive(row.latest))) void loadRows() }, 5000)
})
onUnmounted(() => { loadVersion++; reportVersion++; clearInterval(poll); clearTimeout(queryTimer) })
</script>
