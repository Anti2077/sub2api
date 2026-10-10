<template>
  <AppLayout>
    <div class="space-y-4 pb-8">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.modelIdentity.overviewTitle') }}</h1>
          <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.overviewDescription') }}</p>
        </div>
        <div class="flex flex-wrap gap-2"><button class="btn btn-primary min-h-11" type="button" @click="openAddPlan"><Icon name="plus" size="sm" />{{ t('admin.modelIdentity.addPlan') }}</button><button class="btn btn-secondary min-h-11" type="button" :disabled="loading" @click="loadRows()"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button></div>
      </header>
      <details class="card" :open="settingsOpen" @toggle="settingsOpen = ($event.target as HTMLDetailsElement).open">
        <summary class="flex min-h-12 cursor-pointer items-center justify-between gap-3 px-4 py-3 text-sm font-semibold"><span>{{ t('admin.modelIdentity.connectionSettings') }}</span><span class="text-xs font-normal text-gray-500">{{ publicBaseURL !== savedPublicBaseURL ? t('admin.modelIdentity.settingsUnsaved') : savedPublicBaseURL ? t('admin.modelIdentity.settingsConfigured') : '' }}</span></summary>
        <div class="border-t border-gray-200 p-4 dark:border-dark-700">
        <form class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end" @submit.prevent="saveConnectionSettings">
          <div class="min-w-0 flex-1">
            <label for="identity-base-url" class="text-sm font-medium">{{ t('admin.modelIdentity.publicBaseURL') }}</label>
            <input id="identity-base-url" v-model="publicBaseURL" type="url" required maxlength="2048" class="input mt-1 min-h-11 w-full" placeholder="https://your-site.example.com" :disabled="settingsLoading || settingsSaving" :aria-invalid="Boolean(settingsError)" aria-describedby="identity-base-url-help identity-base-url-feedback" @input="settingsSaved = false; settingsOpen = true; settingsError = ''" />
            <p id="identity-base-url-help" class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.publicBaseURLHint') }}</p>
          </div>
          <button type="submit" class="btn btn-primary min-h-11 shrink-0" :disabled="settingsLoading || settingsSaving || !publicBaseURL.trim()">{{ settingsSaving ? t('common.saving') : t('common.save') }}</button>
        </form>
        <p id="identity-base-url-feedback" :role="settingsError ? 'alert' : 'status'" class="mt-2 text-sm" :class="settingsError ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-300'">{{ settingsError || (settingsSaved ? t('admin.modelIdentity.settingsSaved') : (!settingsLoading && !savedPublicBaseURL ? t('admin.modelIdentity.publicBaseURLRequired') : '')) }}</p>
        </div>
      </details>
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
        <div v-else-if="!filteredRows.length" class="p-8 text-center"><p class="text-sm text-gray-500">{{ t(total || query || statusFilter !== 'all' ? 'common.noData' : 'admin.modelIdentity.emptyPlans') }}</p><button class="btn btn-primary mt-4 min-h-11" type="button" @click="openAddPlan">{{ t('admin.modelIdentity.addPlan') }}</button></div>
        <div v-else class="hidden overflow-x-auto lg:block">
          <table class="w-full min-w-[1060px] table-fixed text-left text-sm">
            <colgroup><col class="w-[120px]" /><col class="w-[150px]" /><col class="w-[160px]" /><col class="w-[300px]" /><col class="w-[120px]" /><col class="w-[90px]" /><col class="w-[120px]" /></colgroup>
            <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-900/50 dark:text-gray-400"><tr>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.account') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.expectedModel') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.lastResult') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.recentRuns') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.nextRun') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('admin.modelIdentity.keyStatus') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('common.actions') }}</th>
            </tr></thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <template v-for="row in filteredRows" :key="row.plan.id">
                <tr :data-plan-id="row.plan.id" class="align-top hover:bg-gray-50/80 dark:hover:bg-dark-800/60">
                  <td class="px-3 py-2.5"><strong class="block text-gray-900 dark:text-white">{{ row.account.name }}</strong><span class="mt-1 block text-xs text-gray-500">#{{ row.account.id }} · {{ groupName(row) }} · {{ t('admin.modelIdentity.planNumber', { id: row.plan.id }) }}</span><p v-if="row.error" role="alert" class="mt-2 max-w-64 text-xs text-red-600 dark:text-red-400">{{ row.error }}</p></td>
                  <td class="px-3 py-2.5"><code class="block break-all text-xs">{{ row.plan.expected_model }}</code><span class="mt-1 block text-xs text-gray-500">{{ row.plan.request_model }}</span></td>
                  <td class="px-3 py-2.5"><span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="identityStatusClass(row.status)">{{ statusLabel(row.status) }}</span><span class="mt-2 block break-all text-xs">{{ row.latest?.report?.detected_model || '—' }}</span><time class="mt-1 block whitespace-nowrap text-xs text-gray-500">{{ formatTime(row.latest?.created_at) }}</time></td>
                  <td class="px-3 py-2.5"><IdentityRecentRuns :runs="row.history.slice(0, 5)" compact @report="openReport" /></td>
                  <td class="px-3 py-2.5 text-xs"><time>{{ formatTime(nextRun(row)) }}</time><span class="mt-1 block text-gray-500">{{ row.plan.enabled ? t('admin.modelIdentity.scheduled') : t('admin.modelIdentity.paused') }}</span></td>
                  <td class="max-w-48 px-3 py-2.5 text-xs"><span :class="row.config?.configuration_error ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-300'">{{ row.config ? (row.config.configuration_error ? t('admin.modelIdentity.keyError') : t('admin.modelIdentity.keyReady')) : t('admin.modelIdentity.notConfigured') }}</span><span v-if="row.config" class="mt-1 block break-words" :title="row.config.key_name">#{{ row.config.api_key_id }}</span></td>
                  <td class="px-3 py-2.5"><div class="flex flex-col gap-1"><button class="btn btn-primary min-h-11 text-xs" :disabled="Boolean(actionBusy) || accountHasActiveRun(row.account.id)" :aria-busy="actionBusy === row.plan.id" type="button" @click="runPlan(row.plan)"><Icon name="play" size="sm" />{{ actionBusy === row.plan.id ? t('common.loading') : t('admin.modelIdentity.manualTrigger') }}</button><button class="btn btn-secondary min-h-11 text-xs" type="button" @click="editPlan(row)">{{ t('common.edit') }}</button><button class="btn btn-ghost min-h-11 text-xs" type="button" :aria-expanded="expandedRows.has(row.plan.id)" :aria-controls="`identity-history-${row.plan.id}`" @click="toggleHistory(row.plan.id)">{{ expandedRows.has(row.plan.id) ? t('common.collapse') : t('admin.modelIdentity.history') }}</button></div></td>
                </tr>
                <tr v-if="expandedRows.has(row.plan.id)" :id="`identity-history-${row.plan.id}`"><td colspan="7" class="bg-gray-50/70 px-3 py-2.5 dark:bg-dark-900/30"><div class="space-y-3">
                  <IdentityHistoryTable :runs="row.history" :busy="Boolean(actionBusy)" @report="openReport" @cancel="cancelRun" />
                </div></td></tr>
              </template>
            </tbody>
          </table>
        </div>
        <div class="divide-y divide-gray-200 dark:divide-dark-700 lg:hidden">
          <article v-for="row in filteredRows" :key="row.plan.id" class="space-y-4 p-4" :data-plan-id="row.plan.id">
            <div class="flex items-start justify-between gap-3"><div><h3 class="font-semibold text-gray-900 dark:text-white">{{ row.account.name }}</h3><p class="mt-1 text-xs text-gray-500">#{{ row.account.id }} · {{ groupName(row) }} · {{ t('admin.modelIdentity.planNumber', { id: row.plan.id }) }}</p></div><span class="shrink-0 rounded-full px-2 py-1 text-xs" :class="identityStatusClass(row.status)">{{ statusLabel(row.status) }}</span></div>
            <p v-if="row.error" role="alert" class="text-sm text-red-600">{{ row.error }}</p>
            <dl class="grid grid-cols-2 gap-3 text-xs"><div><dt class="text-gray-500">{{ t('admin.modelIdentity.expectedModel') }}</dt><dd class="mt-1 break-all">{{ row.plan.expected_model }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.detectedModel') }}</dt><dd class="mt-1 break-all">{{ row.latest?.report?.detected_model || '—' }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.nextRun') }}</dt><dd class="mt-1">{{ formatTime(nextRun(row)) }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.keyStatus') }}</dt><dd class="mt-1">{{ row.config ? (row.config.configuration_error ? t('admin.modelIdentity.keyError') : `#${row.config.api_key_id}`) : t('admin.modelIdentity.notConfigured') }}</dd></div></dl>
            <div><h4 class="mb-2 text-xs text-gray-500">{{ t('admin.modelIdentity.recentRuns') }}</h4><IdentityRecentRuns :runs="row.history.slice(0, 5)" @report="openReport" /></div>
            <div class="flex flex-wrap gap-2"><button class="btn btn-secondary min-h-11" type="button" @click="editPlan(row)">{{ t('common.edit') }}</button><button class="btn btn-primary min-h-11 text-xs" :disabled="Boolean(actionBusy) || accountHasActiveRun(row.account.id)" :aria-busy="actionBusy === row.plan.id" type="button" @click="runPlan(row.plan)"><Icon name="play" size="sm" />{{ actionBusy === row.plan.id ? t('common.loading') : t('admin.modelIdentity.manualTrigger') }}</button><button v-if="isActive(row.latest)" class="btn btn-secondary min-h-11" :disabled="Boolean(actionBusy)" type="button" @click="cancelRun(row.latest!)">{{ t('common.cancel') }}</button></div>
          </article>
        </div>
        <Pagination v-if="total > pageSize" :page="page" :page-size="pageSize" :total="total" :show-page-size-selector="false" @update:page="changePage" />
      </section>
      <section class="grid gap-6 lg:grid-cols-3">
        <div class="card p-5 lg:col-span-2"><h2 class="font-semibold">{{ t('admin.modelIdentity.recentActivity') }}</h2><p class="mt-1 text-xs text-gray-500">{{ t('admin.modelIdentity.currentPage') }}</p><p v-if="!events.length" class="mt-5 text-sm text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p><button v-for="event in events" :key="event.run.id" type="button" class="mt-3 flex min-h-11 w-full flex-wrap items-center justify-between gap-2 rounded-lg border border-gray-100 p-3 text-left dark:border-dark-700" @click="openReport(event.run)"><span class="text-sm">{{ event.name }} <span class="ml-2 rounded-full px-2 py-1 text-xs" :class="identityStatusClass(identityRunStatus(event.run))">{{ statusLabel(identityRunStatus(event.run)) }}</span><span class="mt-1 block break-all text-xs text-gray-500">{{ event.run.request_model }} → {{ event.run.report?.detected_model || '—' }}</span></span><time class="text-xs text-gray-500">{{ formatTime(event.run.created_at) }}</time></button></div>
        <div class="card p-5"><h2 class="font-semibold">{{ t('admin.modelIdentity.scheduleSummary') }}</h2><dl class="mt-5 space-y-4 text-sm"><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('admin.modelIdentity.enabledPlans') }}</dt><dd>{{ enabledPlans.length }}</dd></div><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('admin.modelIdentity.runningNow') }}</dt><dd>{{ rows.filter(row => isActive(row.latest)).length }}</dd></div><div><dt class="text-gray-500">{{ t('admin.modelIdentity.nextWindow') }}</dt><dd class="mt-1">{{ formatTime(scheduleNext) }}</dd></div></dl><p class="mt-5 text-xs text-gray-500">{{ t('admin.modelIdentity.scheduleHint') }}</p></div>
      </section>
      <BaseDialog :show="addPlanOpen" :title="t(editingPlanID ? 'admin.modelIdentity.editPlan' : 'admin.modelIdentity.addPlan')" width="normal" :close-on-escape="!addSaving" :show-close-button="!addSaving" @close="closeAddPlan">
        <form class="space-y-4" @submit.prevent="saveNewPlan">
          <div><label class="block text-sm" for="identity-add-account">{{ t('admin.modelIdentity.account') }}</label><Select id="identity-add-account" v-model="addAccountID" class="mt-1" :options="addAccountOptions" :aria-label="t('admin.modelIdentity.account')" :search-placeholder="t('admin.modelIdentity.filterAccounts')" searchable remote :loading="accountsLoading" :disabled="addBusy || Boolean(editingPlanID)" @search="searchAddAccounts" /></div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div><label class="text-sm" for="identity-add-user">{{ t('admin.modelIdentity.testUser') }}</label><Select id="identity-add-user" v-model="addUserID" class="mt-1" :options="addUserOptions" :aria-label="t('admin.modelIdentity.testUser')" :search-placeholder="t('admin.modelIdentity.searchUser')" searchable remote :loading="usersLoading" :disabled="addBusy || !addAccountID" @search="searchAddUsers" /></div>
            <label class="text-sm">{{ t('admin.modelIdentity.billingGroup') }}<select v-model="addGroupID" class="input mt-1 min-h-11 w-full" :disabled="addBusy || !addAccountID" required><option :value="0" disabled>{{ t('common.select') }}</option><option v-for="group in addCompatibleGroups" :key="group.id" :value="group.id">{{ group.name }}</option></select></label>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-sm">{{ t('admin.modelIdentity.requestModel') }}<input id="identity-add-model" v-model="addDraft.request_model" required maxlength="256" class="input mt-1 min-h-11 w-full" /></label>
            <label class="text-sm">{{ t('admin.modelIdentity.expectedModel') }}<select v-model="addDraft.expected_model" required class="input mt-1 min-h-11 w-full"><option value="" disabled>{{ t('common.select') }}</option><option v-for="model in addCatalog" :key="model.id" :value="model.id">{{ model.name }}</option></select></label>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-sm">{{ t('admin.modelIdentity.interval') }}<input v-model.number="addDraft.interval_minutes" type="number" min="15" max="10080" required class="input mt-1 min-h-11 w-full" /></label>
            <label class="flex min-h-11 items-center gap-2 text-sm"><input v-model="addDraft.enabled" type="checkbox" />{{ t('admin.modelIdentity.scheduled') }}</label>
          </div>
          <p v-if="accountsLoading || addBusy" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p><p v-if="addError" role="alert" class="text-sm text-red-600">{{ addError }}</p>
          <p class="text-sm text-gray-500">{{ t('admin.modelIdentity.addPlanHint') }}</p>
          <p v-if="addConfig" class="text-xs text-gray-500">{{ t('admin.modelIdentity.sharedConfigHint') }}</p>
          <p class="text-xs text-gray-500">{{ t('admin.modelIdentity.remoteConsent') }}</p>
          <div class="flex justify-end gap-2"><button v-if="editingPlanID" class="btn btn-ghost min-h-11 mr-auto text-red-600" type="button" :disabled="addBusy" @click="deleteCurrentPlan">{{ t('common.delete') }}</button><button class="btn btn-secondary min-h-11" type="button" :disabled="addBusy" @click="closeAddPlan">{{ t('common.cancel') }}</button><button class="btn btn-primary min-h-11" type="submit" :disabled="addBusy || accountsLoading || !addAccountID || !addUserID || !addGroupID || !addDraft.request_model.trim() || !addDraft.expected_model">{{ addBusy ? t('common.saving') : t(editingPlanID ? 'common.save' : 'admin.modelIdentity.addPlan') }}</button></div>
        </form>
      </BaseDialog>
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
import Select from '@/components/common/Select.vue'
import IdentityRecentRuns from '@/components/admin/account/IdentityRecentRuns.vue'
import IdentityHistoryTable from '@/components/admin/account/IdentityHistoryTable.vue'
import { adminAPI } from '@/api/admin'
import * as identityAPI from '@/api/admin/modelIdentity'
import { identityRunStatus, identityStatusClass, isActive, type IdentityStatus } from '@/utils/modelIdentity'
import type { Account, AccountListItem, AdminUser, Group } from '@/types'

type IdentityRow = { account: AccountListItem; config: identityAPI.IdentityConfig | null; plans: identityAPI.IdentityPlan[]; history: identityAPI.IdentityRun[]; latest?: identityAPI.IdentityRun; status: IdentityStatus; error: string }
const { t, locale } = useI18n()
const rows = ref<IdentityRow[]>([]), groups = ref<Group[]>([])
const addPlanOpen = ref(false), editingPlanID = ref(0), addAccountID = ref(0), accountSearch = ref(''), addError = ref(''), accountsLoading = ref(false), usersLoading = ref(false), addBusy = ref(false), addAccounts = ref<AccountListItem[]>([])
const addSaving = ref(false)
const addAccount = ref<Account | null>(null), addConfig = ref<identityAPI.IdentityConfig | null>(null), addUsers = ref<AdminUser[]>([]), addCatalog = ref<identityAPI.IdentityModel[]>([])
const addUserID = ref(0), addGroupID = ref(0), addDraft = ref({ request_model: '', expected_model: '', interval_minutes: 120, enabled: false })
let accountSearchVersion = 0, addFormVersion = 0, userSearchVersion = 0
const loading = ref(false), loadError = ref(''), query = ref(''), statusFilter = ref('all')
const publicBaseURL = ref(''), settingsLoading = ref(true), settingsSaving = ref(false), settingsError = ref(''), settingsSaved = ref(false)
const savedPublicBaseURL = ref(''), settingsOpen = ref(true)
const page = ref(1), total = ref(0), pageSize = 20
const expandedRows = ref(new Set<number>()), actionBusy = ref(0)
const reportOpen = ref(false), reportLoading = ref(false), reportError = ref(''), selectedReport = ref<identityAPI.IdentityRun | null>(null)
let loadVersion = 0, reportVersion = 0, queryTimer: ReturnType<typeof setTimeout> | undefined, poll: ReturnType<typeof setInterval> | undefined
const filterStatuses: IdentityStatus[] = ['matched', 'mismatched', 'inconclusive', 'running', 'queued', 'configuration_error', 'request_error', 'service_error', 'notConfigured', 'untested']
const planRows = computed(() => rows.value.flatMap(row => row.plans.map(plan => {
  const history = row.history.filter(run => run.plan_id === plan.id)
  return { ...row, plan, history, latest: history[0], status: row.status === 'service_error' ? 'service_error' as IdentityStatus : row.config?.configuration_error ? 'configuration_error' as IdentityStatus : history[0] ? identityRunStatus(history[0]) : 'untested' as IdentityStatus }
})))
const filteredRows = computed(() => planRows.value.filter(row => statusFilter.value === 'all' || row.status === statusFilter.value))
const addAccountOptions = computed(() => addAccounts.value.map(account => ({ value: account.id, label: `${account.name} (#${account.id})` })))
const addUserOptions = computed(() => addUsers.value.map(user => ({ value: user.id, label: `${user.username || user.email} (#${user.id})` })))
const addCompatibleGroups = computed(() => groups.value.filter(group => addAccount.value?.group_ids?.includes(group.id) && group.platform === addAccount.value?.platform && !group.claude_code_only))
const enabledPlans = computed(() => planRows.value.map(row => row.plan).filter(plan => plan.enabled))
const scheduleNext = computed(() => enabledPlans.value.map(plan => plan.next_run_at).filter((value): value is string => Boolean(value)).sort()[0])
const stats = computed(() => [
  { label: t('admin.modelIdentity.totalPlans'), value: planRows.value.length },
  { label: t('admin.modelIdentity.matchedAccounts'), value: planRows.value.filter(row => row.status === 'matched').length },
  { label: t('admin.modelIdentity.attentionNeeded'), value: planRows.value.filter(row => ['mismatched', 'inconclusive', 'configuration_error', 'request_error', 'service_error', 'timed_out'].includes(row.status)).length },
  { label: t('admin.modelIdentity.enabledPlans'), value: enabledPlans.value.length },
])
const events = computed(() => planRows.value.flatMap(row => row.history.map(run => ({ name: row.account.name, run }))).sort((a, b) => +new Date(b.run.created_at) - +new Date(a.run.created_at)).slice(0, 5))
const reportReason = computed(() => {
  const assessment = selectedReport.value?.report?.identityAssessment as { verdict?: { reasoning?: string } } | undefined
  return assessment?.verdict?.reasoning || ''
})
function statusLabel(status: IdentityStatus) { return t(`admin.modelIdentity.${status}`) }
function message(error: unknown) { return error && typeof error === 'object' && 'message' in error ? String(error.message) : String(error) }
async function loadConnectionSettings() {
  settingsLoading.value = true; settingsError.value = ''
  try { publicBaseURL.value = savedPublicBaseURL.value = (await identityAPI.settings()).public_base_url; settingsOpen.value = !savedPublicBaseURL.value }
  catch (error) { settingsError.value = message(error); settingsOpen.value = true }
  finally { settingsLoading.value = false }
}
async function saveConnectionSettings() {
  if (settingsLoading.value || settingsSaving.value || !publicBaseURL.value.trim()) return
  settingsSaving.value = true; settingsError.value = ''; settingsSaved.value = false
  try { publicBaseURL.value = savedPublicBaseURL.value = (await identityAPI.saveSettings({ public_base_url: publicBaseURL.value.trim() })).public_base_url; settingsSaved.value = true }
  catch (error) { settingsError.value = message(error); settingsOpen.value = true }
  finally { settingsSaving.value = false }
}
function formatTime(value?: string) { return value ? new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US', { hour12: false }) : '—' }
function groupName(row: IdentityRow) { return row.config ? groups.value.find(group => group.id === row.config?.group_id)?.name || `#${row.config.group_id}` : t('admin.modelIdentity.notConfigured') }
function nextRun(row: { plan: identityAPI.IdentityPlan }) { return row.plan.enabled ? row.plan.next_run_at : undefined }
function toggleHistory(id: number) { const next = new Set(expandedRows.value); next.has(id) ? next.delete(id) : next.add(id); expandedRows.value = next }
async function loadRow(account: AccountListItem): Promise<IdentityRow> {
  const row: IdentityRow = { account, config: null, plans: [], history: [], status: 'notConfigured', error: '' }
  try {
    try { row.config = await identityAPI.config(account.id) } catch (error) { const failure = error as { status?: number; response?: { status?: number } }; if ((failure.status ?? failure.response?.status) !== 404) throw error }
    if (!row.config) return row
    row.plans = await identityAPI.plans(account.id)
    const histories: identityAPI.IdentityRun[] = []
    for (const plan of row.plans) histories.push(...await identityAPI.history(plan.id))
    row.history = histories.sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at))
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
    const result = await identityAPI.plannedAccounts(page.value, pageSize, query.value)
    if (version !== loadVersion) return
    const lastPage = Math.max(1, Math.ceil(result.total / pageSize))
    if (page.value > lastPage) { page.value = lastPage; await loadRows(); return }
    const loaded: IdentityRow[] = []
    let cursor = 0
    await Promise.all(Array.from({ length: Math.min(4, result.account_ids.length) }, async () => {
      while (cursor < result.account_ids.length) { const account = await adminAPI.accounts.getById(result.account_ids[cursor++]); const row = await loadRow(account); if (version !== loadVersion) return; loaded.push(row) }
    }))
    if (version !== loadVersion) return
    const positions = new Map(result.account_ids.map((id, index) => [id, index]))
    rows.value = loaded.sort((a, b) => (positions.get(a.account.id) || 0) - (positions.get(b.account.id) || 0)); total.value = result.total
  } catch (error) { if (version === loadVersion) { rows.value = []; loadError.value = message(error) } }
  finally { if (version === loadVersion) loading.value = false }
}
async function searchAddAccountsNow() {
  const version = ++accountSearchVersion
  accountsLoading.value = true; addError.value = ''
  try { const result = await adminAPI.accounts.list(1, 50, { lite: 'true', search: accountSearch.value }); if (version === accountSearchVersion) { addAccounts.value = result.items; if (addAccount.value && !result.items.some(account => account.id === addAccountID.value)) addAccounts.value.push(addAccount.value) } }
  catch (error) { if (version === accountSearchVersion) addError.value = message(error) }
  finally { if (version === accountSearchVersion) accountsLoading.value = false }
}
function searchAddAccounts(search: string) { accountSearch.value = search; void searchAddAccountsNow() }
function openAddPlan() { closeAddPlan(); accountSearch.value = ''; addAccounts.value = []; addPlanOpen.value = true; void searchAddAccountsNow() }
function closeAddPlan() { if (addSaving.value) return; addPlanOpen.value = false; accountSearchVersion++; addFormVersion++; userSearchVersion++; editingPlanID.value = 0; addAccountID.value = 0; addBusy.value = false; accountsLoading.value = false; usersLoading.value = false; addAccount.value = null; addConfig.value = null; addUsers.value = []; addCatalog.value = []; addUserID.value = 0; addGroupID.value = 0; addError.value = ''; addDraft.value = { request_model: '', expected_model: '', interval_minutes: 120, enabled: false } }
async function searchAddUsers(search: string) {
  const version = ++userSearchVersion
  usersLoading.value = true
  try { const result = await adminAPI.users.list(1, 20, { status: 'active', search }); if (version === userSearchVersion) { const selected = addUsers.value.find(user => user.id === addUserID.value); addUsers.value = result.items; if (selected && !result.items.some(user => user.id === selected.id)) addUsers.value.push(selected) } }
  catch (error) { if (version === userSearchVersion) addError.value = message(error) }
  finally { if (version === userSearchVersion) usersLoading.value = false }
}
async function loadAddAccount() {
  const version = ++addFormVersion
  userSearchVersion++; usersLoading.value = false
  const id = addAccountID.value
  addAccount.value = null; addConfig.value = null; addUserID.value = 0; addGroupID.value = 0
  if (!id) { addBusy.value = false; return }
  addBusy.value = true; addError.value = ''
  try {
    const [account, userResult, modelResult] = await Promise.all([adminAPI.accounts.getById(id), adminAPI.users.list(1, 20, { status: 'active' }), identityAPI.models()])
    if (version !== addFormVersion) return
    addAccount.value = account
    addUsers.value = userResult.items
    addCatalog.value = modelResult.models
    try {
      const config = await identityAPI.config(id)
      if (version !== addFormVersion) return
      addConfig.value = config; addUserID.value = config.user_id; addGroupID.value = config.group_id
      if (!addUsers.value.some(user => user.id === config.user_id)) { const user = await adminAPI.users.getById(config.user_id); if (version === addFormVersion) addUsers.value.push(user) }
    } catch (error) { const failure = error as { status?: number; response?: { status?: number } }; if ((failure.status ?? failure.response?.status) !== 404) throw error }
  } catch (error) { if (version === addFormVersion) { addAccount.value = null; addError.value = message(error) } }
  finally { if (version === addFormVersion) addBusy.value = false }
}
async function saveNewPlan() {
  if (addBusy.value || !addAccount.value || addAccount.value.id !== addAccountID.value || !addUserID.value || !addCompatibleGroups.value.some(group => group.id === addGroupID.value) || !addDraft.value.request_model.trim() || !addCatalog.value.some(model => model.id === addDraft.value.expected_model)) return
  addBusy.value = true; addSaving.value = true; addError.value = ''
  const accountID = addAccount.value.id, planID = editingPlanID.value
  const configuration = { user_id: addUserID.value, group_id: addGroupID.value }
  const payload = { account_id: accountID, ...addDraft.value, request_model: addDraft.value.request_model.trim() }
  try {
    if (!addConfig.value || addConfig.value.user_id !== configuration.user_id || addConfig.value.group_id !== configuration.group_id) addConfig.value = await identityAPI.saveConfig(accountID, configuration)
    if (planID) await identityAPI.updatePlan(planID, payload)
    else await identityAPI.savePlan(payload)
    addSaving.value = false; closeAddPlan(); await loadRows()
  }
  catch (error) { addError.value = message(error) }
  finally { addSaving.value = false; addBusy.value = false }
}
watch(addAccountID, () => { if (addPlanOpen.value) void loadAddAccount() })
function changePage(value: number) { page.value = value; void loadRows() }
function editPlan(row: IdentityRow & { plan: identityAPI.IdentityPlan }) {
  closeAddPlan(); addAccounts.value = [row.account]; editingPlanID.value = row.plan.id
  addDraft.value = { request_model: row.plan.request_model, expected_model: row.plan.expected_model, interval_minutes: row.plan.interval_minutes, enabled: row.plan.enabled }
  addPlanOpen.value = true; addAccountID.value = row.account.id
}
async function deleteCurrentPlan() {
  if (addBusy.value || !editingPlanID.value) return
  addBusy.value = true; addSaving.value = true; addError.value = ''
  try { await identityAPI.deletePlan(editingPlanID.value); addSaving.value = false; closeAddPlan(); await loadRows() }
  catch (error) { addError.value = message(error) }
  finally { addSaving.value = false; addBusy.value = false }
}
function accountHasActiveRun(accountID: number) { return rows.value.some(row => row.account.id === accountID && row.history.some(isActive)) }
async function runPlan(plan: identityAPI.IdentityPlan) {
  if (actionBusy.value || accountHasActiveRun(plan.account_id)) return
  loadError.value = ''
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
  await Promise.allSettled([loadConnectionSettings(), loadRows(), adminAPI.groups.getAll().then(result => { groups.value = result }).catch(error => { loadError.value = message(error) })])
  poll = setInterval(() => { if (!document.hidden && !loading.value && !addPlanOpen.value && planRows.value.some(row => isActive(row.latest))) void loadRows() }, 5000)
})
onUnmounted(() => { accountSearchVersion++; addFormVersion++; userSearchVersion++; loadVersion++; reportVersion++; clearInterval(poll); clearTimeout(queryTimer) })
</script>
