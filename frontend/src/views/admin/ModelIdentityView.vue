<template>
  <AppLayout>
    <div class="space-y-4 pb-8">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.modelIdentity.overviewTitle') }}</h1>
          <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.overviewDescription') }}</p>
        </div>
        <div class="flex flex-wrap gap-2"><button data-testid="identity-settings-button" class="btn btn-secondary min-h-11" type="button" @click="openSettings"><Icon name="cog" size="sm" />{{ t('admin.modelIdentity.settingsTitle') }}</button><button class="btn btn-primary min-h-11" type="button" @click="openAddPlan"><Icon name="plus" size="sm" />{{ t('admin.modelIdentity.addPlan') }}</button><button class="btn btn-secondary min-h-11" type="button" :disabled="loading" @click="loadRows()"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button></div>
      </header>

      <BaseDialog :show="settingsOpen" :title="t('admin.modelIdentity.settingsTitle')" width="normal" :close-on-escape="!settingsSaving" :show-close-button="!settingsSaving" @close="closeSettings">
        <form id="identity-settings-form" class="space-y-5" :aria-busy="settingsLoading || settingsSaving" @submit.prevent="saveConnectionSettings" @input="settingsSaved = false; settingsError = ''">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.settingsHint') }}</p>
          <p v-if="settingsLoading" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
          <div>
            <label for="identity-test-user" class="input-label">{{ t('admin.modelIdentity.fixedTestUser') }}</label>
            <Select id="identity-test-user" v-model="settingsDraft.test_user_id" :options="testUserOptions" :aria-label="t('admin.modelIdentity.fixedTestUser')" :search-placeholder="t('admin.modelIdentity.searchUser')" searchable remote :loading="usersLoading" :disabled="settingsLoading || settingsSaving || !savedSettings" @search="searchTestUsers" />
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.fixedTestUserHint') }}</p>
          </div>
          <div class="grid gap-5 sm:grid-cols-2">
            <div>
              <label for="identity-concurrent-accounts" class="input-label">{{ t('admin.modelIdentity.concurrentAccounts') }}</label>
              <input id="identity-concurrent-accounts" v-model.number="settingsDraft.concurrent_accounts" type="number" min="1" max="2" step="1" required class="input min-h-11 w-full" :disabled="settingsLoading || settingsSaving || !savedSettings" aria-describedby="identity-concurrent-accounts-help" />
              <p id="identity-concurrent-accounts-help" class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.concurrentAccountsHint') }}</p>
            </div>
            <div>
              <label for="identity-probe-concurrency" class="input-label">{{ t('admin.modelIdentity.probeConcurrency') }}</label>
              <input id="identity-probe-concurrency" v-model.number="settingsDraft.probe_concurrency" type="number" min="1" max="10" step="1" required class="input min-h-11 w-full" :disabled="settingsLoading || settingsSaving || !savedSettings" aria-describedby="identity-probe-concurrency-help" />
              <p id="identity-probe-concurrency-help" class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.probeConcurrencyHint') }}</p>
            </div>
          </div>
          <p class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">{{ t('admin.modelIdentity.concurrencyHint') }}</p>
          <p id="identity-base-url-feedback" :role="settingsError ? 'alert' : 'status'" class="text-sm" :class="settingsError ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-300'">{{ settingsError || (settingsSaved ? t('admin.modelIdentity.settingsSaved') : settingsDirty ? t('admin.modelIdentity.settingsUnsaved') : savedSettings?.public_base_url ? t('admin.modelIdentity.settingsConfigured') : '') }}</p>
          <div class="flex flex-wrap justify-end gap-2">
            <button v-if="!savedSettings && !settingsLoading" class="btn btn-secondary min-h-11" type="button" @click="loadConnectionSettings">{{ t('common.tryAgain') }}</button>
            <button class="btn btn-secondary min-h-11" type="button" :disabled="settingsSaving" @click="closeSettings">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary min-h-11" type="submit" :disabled="settingsLoading || settingsSaving || !savedSettings || !settingsValid || !settingsDirty">{{ settingsSaving ? t('common.saving') : t('common.save') }}</button>
          </div>
        </form>
      </BaseDialog>
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
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-600">
          <article v-for="row in filteredRows" :key="row.plan.id" class="min-w-0 px-4 py-1.5" :class="selectedPlanID === row.plan.id ? 'bg-primary-50/60 dark:bg-primary-900/15' : 'bg-white dark:bg-dark-800'" :data-plan-id="row.plan.id">
            <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 md:flex-nowrap">
              <button type="button" class="flex min-h-11 min-w-0 flex-1 basis-full flex-wrap items-center gap-x-5 gap-y-1 text-left focus-visible:outline-primary-500 sm:flex-nowrap md:basis-0" :aria-pressed="selectedPlanID === row.plan.id" @click="selectPlan(row.plan.id)">
                <span class="min-w-0 basis-full sm:w-40 sm:shrink-0 sm:basis-auto">
                  <span class="block truncate text-sm font-medium text-gray-900 dark:text-white" :title="`${row.account.name} · ${row.groupName} · ${t('admin.modelIdentity.planNumber', { id: row.plan.id })}`">{{ row.account.name }} <span class="text-xs font-normal text-gray-400">#{{ row.account.id }}</span></span>
                  <span class="sr-only">{{ row.groupName }} · {{ t('admin.modelIdentity.planNumber', { id: row.plan.id }) }}</span>
                </span>
                <span class="min-w-0 flex-1 truncate font-mono text-xs text-gray-600 dark:text-gray-300" :title="row.plan.request_model">{{ row.plan.request_model }}</span>
                <span class="flex min-w-0 flex-1 items-center gap-2">
                  <span class="shrink-0 rounded px-1.5 py-0.5 text-xs" :class="identityStatusClass(row.status)">{{ statusLabel(row.status) }}</span>
                  <span class="min-w-0 truncate text-xs text-gray-600 dark:text-gray-300" :title="identityDetectedModel(row.latest)">{{ identityDetectedModel(row.latest) }}</span>
                </span>
              </button>
              <IdentityRecentRuns :runs="row.history.slice(0, 5)" compact dense @report="openReport" />
              <div class="ml-auto flex shrink-0 items-center gap-1">
                <IdentityScheduleToggle :plan-id="row.plan.id" :enabled="row.plan.enabled" :busy="scheduleBusy === row.plan.id" :disabled="loading || Boolean(scheduleBusy)" compact :title="`${t('admin.modelIdentity.nextRun')}: ${formatTime(nextRun(row))}`" @change="toggleSchedule(row.plan)" />
                <button type="button" class="btn btn-ghost min-h-11 min-w-11 p-2" :title="t('admin.modelIdentity.manualTrigger')" :aria-label="t('admin.modelIdentity.manualTrigger')" :disabled="Boolean(actionBusy) || accountHasActiveRun(row.account.id)" @click="runPlan(row.plan)"><Icon name="play" size="sm" /><span class="sr-only">{{ t('admin.modelIdentity.manualTrigger') }}</span></button>
                <button type="button" class="btn btn-ghost min-h-11 min-w-11 p-2" :title="t('common.edit')" :aria-label="t('common.edit')" @click="editPlan(row)"><Icon name="edit" size="sm" /><span class="sr-only">{{ t('common.edit') }}</span></button>
                <button type="button" class="btn btn-ghost min-h-11 min-w-11 p-2" :title="t('admin.modelIdentity.history')" :aria-label="t('admin.modelIdentity.history')" @click="loadHistory(row.plan)"><Icon name="clock" size="sm" /><span class="sr-only">{{ t('admin.modelIdentity.history') }}</span></button>
              </div>
            </div>
            <p v-if="row.error" role="alert" class="pb-1 text-xs text-red-600 dark:text-red-400">{{ row.error }}</p>
          </article>
        </div>
        <Pagination v-if="total > pageSize" :page="page" :page-size="pageSize" :total="total" :show-page-size-selector="false" @update:page="changePage" />
      </section>
      <section class="space-y-4" aria-live="polite" aria-atomic="false">
        <div class="flex flex-wrap items-center justify-between gap-3"><div><h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.modelIdentity.currentReport') }}</h2><p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.selectionHint') }}</p></div><button v-if="isActive(selectedReport)" type="button" class="btn btn-secondary min-h-11" :disabled="Boolean(actionBusy)" @click="cancelRun(selectedReport!)">{{ t('common.cancel') }}</button></div>
        <p v-if="reportLoading" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p><p v-if="reportError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ reportError }}</p>
        <FingerpointReport v-if="selectedReport?.report?.engine === 'lm-fingerpoint-detector'" :run="selectedReport" />
        <div v-else-if="selectedReport" class="card space-y-4 p-5"><div class="flex flex-wrap justify-between gap-2"><p>{{ statusLabel(identityRunStatus(selectedReport)) }} · {{ formatTime(selectedReport.created_at) }}</p><IdentityOfficialLink :run="selectedReport" /></div><IdentityScopeSummary :run="selectedReport" /><IdentityPerformance :report="selectedReport.report" /><IdentityModelVotes :report="selectedReport.report" /><details><summary class="min-h-11 cursor-pointer">{{ t('admin.modelIdentity.evidence') }}</summary><pre class="max-h-96 overflow-auto whitespace-pre-wrap break-words text-xs">{{ JSON.stringify(selectedReport.report, null, 2) }}</pre></details></div>
        <p v-else-if="!reportLoading" class="card p-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.noHistory') }}</p>
      </section>
      <BaseDialog :show="historyOpen" :title="t('admin.modelIdentity.history')" width="wide" @close="historyOpen = false"><IdentityHistoryTable :runs="historyRuns" :busy="Boolean(actionBusy)" @report="openReport" @cancel="cancelRun" /></BaseDialog>
      <BaseDialog :show="addPlanOpen" :title="t(editingPlanID ? 'admin.modelIdentity.editPlan' : 'admin.modelIdentity.addPlan')" width="normal" :close-on-escape="!addSaving" :show-close-button="!addSaving" @close="closeAddPlan">
        <form class="space-y-4" @submit.prevent="saveNewPlan">
          <div><label class="block text-sm" for="identity-add-account">{{ t('admin.modelIdentity.account') }}</label><Select id="identity-add-account" v-model="addAccountID" class="mt-1" :options="addAccountOptions" :aria-label="t('admin.modelIdentity.account')" :search-placeholder="t('admin.modelIdentity.filterAccounts')" searchable remote :loading="accountsLoading" :disabled="addBusy || Boolean(editingPlanID)" @search="searchAddAccounts" /></div>
          <div class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-900">
            <p>{{ t('admin.modelIdentity.fixedTestUser') }} · {{ fixedTestUserLabel }}</p>
            <button v-if="!savedSettings?.test_user_id" type="button" class="btn btn-secondary mt-2 min-h-11" @click="openSettings">{{ t('admin.modelIdentity.configureFixedTestUser') }}</button>
          </div>
          <label class="block text-sm">{{ t('admin.modelIdentity.billingGroup') }}<select v-model="addGroupID" class="input mt-1 min-h-11 w-full" :disabled="addBusy || !addAccountID" required><option :value="0" disabled>{{ t('common.select') }}</option><option v-for="group in addCompatibleGroups" :key="group.id" :value="group.id">{{ group.name }}</option></select></label>
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-sm">{{ t('admin.modelIdentity.requestModel') }}<input id="identity-add-model" v-model="addDraft.request_model" required maxlength="256" class="input mt-1 min-h-11 w-full" /></label>
            <label class="text-sm">{{ t('admin.modelIdentity.expectedModel') }}<input v-model="addDraft.expected_model" list="identity-expected-models" required maxlength="256" class="input mt-1 min-h-11 w-full" /><datalist id="identity-expected-models"><option v-for="model in addCatalog" :key="model.id" :value="model.id">{{ model.name }}</option></datalist></label>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="text-sm">{{ t('admin.modelIdentity.interval') }}<input v-model.number="addDraft.interval_minutes" type="number" min="15" max="10080" required class="input mt-1 min-h-11 w-full" /></label>
            <label class="flex min-h-11 items-center gap-2 text-sm"><input v-model="addDraft.enabled" type="checkbox" />{{ t('admin.modelIdentity.scheduled') }}</label>
          </div>
          <IdentityOptionsForm v-model="addDraft.options" />
          <p v-if="addDraft.expected_model && !addCatalog.some(model => model.id === addDraft.expected_model)" role="status" class="text-sm text-amber-700 dark:text-amber-300">{{ t('admin.modelIdentity.unknownExpectedHint') }}</p>
          <p v-if="accountsLoading || addBusy" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p><p v-if="addError" role="alert" class="text-sm text-red-600">{{ addError }}</p>
          <p class="text-sm text-gray-500">{{ t('admin.modelIdentity.addPlanHint') }}</p>
          <p v-if="addConfig" class="text-xs text-gray-500">{{ t('admin.modelIdentity.sharedConfigHint') }}</p>
          <p class="text-xs text-gray-500">{{ t('admin.modelIdentity.remoteConsent') }}</p>
          <div class="flex justify-end gap-2"><button v-if="editingPlanID" class="btn btn-ghost min-h-11 mr-auto text-red-600" type="button" :disabled="addBusy" @click="deleteCurrentPlan">{{ t('common.delete') }}</button><button class="btn btn-secondary min-h-11" type="button" :disabled="addBusy" @click="closeAddPlan">{{ t('common.cancel') }}</button><button class="btn btn-primary min-h-11" type="submit" :disabled="addBusy || accountsLoading || !addAccountID || !savedSettings?.test_user_id || !addGroupID || !addDraft.request_model.trim() || !addDraft.expected_model">{{ addBusy ? t('common.saving') : t(editingPlanID ? 'common.save' : 'admin.modelIdentity.addPlan') }}</button></div>
        </form>
      </BaseDialog>

    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IdentityScheduleToggle from '@/components/admin/account/IdentityScheduleToggle.vue'
import IdentityOfficialLink from '@/components/admin/account/IdentityOfficialLink.vue'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import IdentityRecentRuns from '@/components/admin/account/IdentityRecentRuns.vue'
import IdentityHistoryTable from '@/components/admin/account/IdentityHistoryTable.vue'
import IdentityPerformance from '@/components/admin/account/IdentityPerformance.vue'
import IdentityScopeSummary from '@/components/admin/account/IdentityScopeSummary.vue'
import IdentityModelVotes from '@/components/admin/account/IdentityModelVotes.vue'
import FingerpointReport from '@/components/model-identity/FingerpointReport.vue'
import IdentityOptionsForm from '@/components/model-identity/IdentityOptionsForm.vue'
import { adminAPI } from '@/api/admin'
import * as identityAPI from '@/api/admin/modelIdentity'
import { identityDetectedModel, identityRunStatus, identityStatusClass, isActive, type IdentityStatus } from '@/utils/modelIdentity'
import type { Account, AccountListItem, AdminUser, Group } from '@/types'

type IdentityRow = { account: Pick<AccountListItem, 'id' | 'name'>; groupName: string; config: identityAPI.IdentityConfig | null; plans: identityAPI.IdentityPlan[]; history: identityAPI.IdentityRun[]; latest?: identityAPI.IdentityRun; status: IdentityStatus; error: string }
const { t, locale } = useI18n()
const rows = ref<IdentityRow[]>([]), groups = ref<Group[]>([])
const addPlanOpen = ref(false), editingPlanID = ref(0), addAccountID = ref(0), accountSearch = ref(''), addError = ref(''), accountsLoading = ref(false), usersLoading = ref(false), addBusy = ref(false), addAccounts = ref<Pick<AccountListItem, 'id' | 'name'>[]>([])
const addSaving = ref(false)
const addAccount = ref<Account | null>(null), addConfig = ref<identityAPI.IdentityConfig | null>(null), testUsers = ref<AdminUser[]>([]), addCatalog = ref<identityAPI.IdentityModel[]>([])
const addGroupID = ref(0), addDraft = ref({ request_model: '', expected_model: '', interval_minutes: 120, enabled: false, options: identityAPI.defaultIdentityOptions() })
let accountSearchVersion = 0, addFormVersion = 0, userSearchVersion = 0
const loading = ref(false), loadError = ref(''), query = ref(''), statusFilter = ref('all')
const settingsLoading = ref(true), settingsSaving = ref(false), settingsError = ref(''), settingsSaved = ref(false), settingsOpen = ref(false)
const defaultSettings: identityAPI.IdentitySettings = { public_base_url: '', concurrent_accounts: 1, probe_concurrency: 4, test_user_id: 0 }
const settingsDraft = ref<identityAPI.IdentitySettings>({ ...defaultSettings })
const savedSettings = ref<identityAPI.IdentitySettings | null>(null)
const settingsDirty = computed(() => Boolean(savedSettings.value) && JSON.stringify(settingsDraft.value) !== JSON.stringify(savedSettings.value))
const settingsValid = computed(() => {
  const value = settingsDraft.value
  return Number.isSafeInteger(value.test_user_id) && value.test_user_id > 0 && Number.isInteger(value.concurrent_accounts) && value.concurrent_accounts >= 1 && value.concurrent_accounts <= 2 && Number.isInteger(value.probe_concurrency) && value.probe_concurrency >= 1 && value.probe_concurrency <= 10
})
const page = ref(1), total = ref(0), pageSize = 20
const actionBusy = ref(0), scheduleBusy = ref(0)
const selectedPlanID = ref(0), historyOpen = ref(false), historyRuns = ref<identityAPI.IdentityRun[]>([])
const reportLoading = ref(false), reportError = ref(''), selectedReport = ref<identityAPI.IdentityRun | null>(null)
let loadVersion = 0, reportVersion = 0, queryTimer: ReturnType<typeof setTimeout> | undefined, poll: ReturnType<typeof setInterval> | undefined
const filterStatuses: IdentityStatus[] = ['ranked', 'partial', 'not_model', 'unscorable', 'interrupted', 'matched', 'mismatched', 'flagshipClass', 'inconclusive', 'running', 'queued', 'configuration_error', 'request_error', 'service_error', 'notConfigured', 'untested']
const planRows = computed(() => rows.value.flatMap(row => row.plans.map(plan => {
  const history = row.history.filter(run => run.plan_id === plan.id)
  return { ...row, plan, history, latest: history[0], status: row.status === 'service_error' ? 'service_error' as IdentityStatus : row.config?.configuration_error ? 'configuration_error' as IdentityStatus : history[0] ? identityRunStatus(history[0]) : 'untested' as IdentityStatus }
})))
const filteredRows = computed(() => planRows.value.filter(row => statusFilter.value === 'all' || row.status === statusFilter.value))
const addAccountOptions = computed(() => addAccounts.value.map(account => ({ value: account.id, label: `${account.name} (#${account.id})` })))
const testUserOptions = computed(() => testUsers.value.map(user => ({ value: user.id, label: `${user.username || user.email} (#${user.id})` })))
const fixedTestUserLabel = computed(() => testUserOptions.value.find(user => user.value === savedSettings.value?.test_user_id)?.label || (savedSettings.value?.test_user_id ? `#${savedSettings.value.test_user_id}` : t('admin.modelIdentity.fixedTestUserRequired')))
const addCompatibleGroups = computed(() => groups.value.filter(group => addAccount.value?.group_ids?.includes(group.id) && group.platform === addAccount.value?.platform && !group.claude_code_only))
const enabledPlans = computed(() => planRows.value.map(row => row.plan).filter(plan => plan.enabled))
const stats = computed(() => [
  { label: t('admin.modelIdentity.totalPlans'), value: planRows.value.length },
  { label: t('admin.modelIdentity.matched'), value: planRows.value.filter(row => row.status === 'matched').length },
  { label: t('admin.modelIdentity.attentionNeeded'), value: planRows.value.filter(row => ['partial', 'not_model', 'unscorable', 'interrupted', 'mismatched', 'flagshipClass', 'inconclusive', 'configuration_error', 'request_error', 'service_error', 'timed_out'].includes(row.status)).length },
  { label: t('admin.modelIdentity.enabledPlans'), value: enabledPlans.value.length },
])
function statusLabel(status: IdentityStatus) { return t(`admin.modelIdentity.${status}`) }
function message(error: unknown) { return error && typeof error === 'object' && 'message' in error ? String(error.message) : String(error) }
function openSettings() {
  if (savedSettings.value) settingsDraft.value = { ...savedSettings.value }
  settingsSaved.value = false
  settingsOpen.value = true
  void searchTestUsers('')
}
function closeSettings() {
  if (settingsSaving.value) return
  settingsOpen.value = false
  if (savedSettings.value) settingsDraft.value = { ...savedSettings.value }
  settingsSaved.value = false
}
async function loadConnectionSettings() {
  settingsLoading.value = true; settingsError.value = ''
  try {
    const settings = { ...defaultSettings, ...await identityAPI.settings() }
    savedSettings.value = { ...settings }
    settingsDraft.value = { ...settings }
    if (settings.test_user_id) {
      const user = await adminAPI.users.getById(settings.test_user_id)
      if (!testUsers.value.some(item => item.id === user.id)) testUsers.value.push(user)
    }
  }
  catch (error) { settingsError.value = message(error) }
  finally { settingsLoading.value = false }
}
async function saveConnectionSettings() {
  if (settingsLoading.value || settingsSaving.value || !savedSettings.value || !settingsValid.value || !settingsDirty.value) return
  settingsSaving.value = true; settingsError.value = ''; settingsSaved.value = false
  try {
    const settings = await identityAPI.saveSettings({ ...settingsDraft.value, public_base_url: settingsDraft.value.public_base_url.trim() })
    savedSettings.value = { ...settings }
    settingsDraft.value = { ...settings }
    settingsSaved.value = true
    await loadRows()
  }
  catch (error) { settingsError.value = message(error) }
  finally { settingsSaving.value = false }
}
function formatTime(value?: string) { return value ? new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US', { hour12: false }) : '—' }
function nextRun(row: { plan: identityAPI.IdentityPlan }) { return row.plan.enabled ? row.plan.next_run_at : undefined }
async function selectPlan(id: number) {
  selectedPlanID.value = id
  const row = planRows.value.find(row => row.plan.id === id)
  const run = row?.history.find(isActive) || row?.history[0]
  if (run) await openReport(run)
  else { reportVersion++; selectedReport.value = null; reportError.value = '' }
}
async function loadHistory(plan: identityAPI.IdentityPlan) {
  try { historyRuns.value = await identityAPI.history(plan.id); historyOpen.value = true }
  catch (error) { loadError.value = message(error) }
}
async function loadRows() {
  const version = ++loadVersion
  loading.value = true; loadError.value = ''
  try {
    const result = await identityAPI.overview(page.value, pageSize, query.value)
    if (version !== loadVersion) return
    const lastPage = Math.max(1, Math.ceil(result.total / pageSize))
    if (page.value > lastPage) { page.value = lastPage; await loadRows(); return }
    rows.value = result.items.map(card => ({ account: { id: card.plan.account_id, name: card.account_name }, groupName: card.group_name, config: card.config, plans: [card.plan], history: card.recent_runs, latest: card.recent_runs[0], status: card.recent_runs[0] ? identityRunStatus(card.recent_runs[0]) : 'untested', error: card.config.configuration_error || '' }))
    total.value = result.total
    const selected = planRows.value.find(row => row.plan.id === selectedPlanID.value)
    if (!selected && planRows.value[0]) await selectPlan(planRows.value[0].plan.id)
    else if (selected?.history[0] && (!selectedReport.value || (isActive(selected.history[0]) && selectedReport.value.id !== selected.history[0].id))) await selectPlan(selected.plan.id)
  } catch (error) { if (version === loadVersion) loadError.value = message(error) }
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
function closeAddPlan() { if (addSaving.value) return; addPlanOpen.value = false; accountSearchVersion++; addFormVersion++; editingPlanID.value = 0; addAccountID.value = 0; addBusy.value = false; accountsLoading.value = false; addAccount.value = null; addConfig.value = null; addCatalog.value = []; addGroupID.value = 0; addError.value = ''; addDraft.value = { request_model: '', expected_model: '', interval_minutes: 120, enabled: false, options: identityAPI.defaultIdentityOptions() } }
async function searchTestUsers(search: string) {
  const version = ++userSearchVersion
  usersLoading.value = true
  try { const result = await adminAPI.users.list(1, 20, { status: 'active', search }); if (version === userSearchVersion) { const selected = testUsers.value.filter(user => user.id === settingsDraft.value.test_user_id || user.id === savedSettings.value?.test_user_id); testUsers.value = result.items; for (const user of selected) if (!testUsers.value.some(item => item.id === user.id)) testUsers.value.push(user) } }
  catch (error) { if (version === userSearchVersion) settingsError.value = message(error) }
  finally { if (version === userSearchVersion) usersLoading.value = false }
}
async function loadAddAccount() {
  const version = ++addFormVersion
  const id = addAccountID.value
  addAccount.value = null; addConfig.value = null; addGroupID.value = 0
  if (!id) { addBusy.value = false; return }
  addBusy.value = true; addError.value = ''
  try {
    const [account, modelResult] = await Promise.all([adminAPI.accounts.getById(id), identityAPI.models()])
    if (version !== addFormVersion) return
    addAccount.value = account
    addCatalog.value = modelResult.models
    try {
      const config = await identityAPI.config(id)
      if (version !== addFormVersion) return
      addConfig.value = config; addGroupID.value = config.group_id
    } catch (error) { const failure = error as { status?: number; response?: { status?: number } }; if ((failure.status ?? failure.response?.status) !== 404) throw error }
  } catch (error) { if (version === addFormVersion) { addAccount.value = null; addError.value = message(error) } }
  finally { if (version === addFormVersion) addBusy.value = false }
}
async function saveNewPlan() {
  if (addBusy.value || !addAccount.value || addAccount.value.id !== addAccountID.value || !savedSettings.value?.test_user_id || !addCompatibleGroups.value.some(group => group.id === addGroupID.value) || !addDraft.value.request_model.trim() || !addDraft.value.expected_model.trim()) return
  addBusy.value = true; addSaving.value = true; addError.value = ''
  const accountID = addAccount.value.id, planID = editingPlanID.value
  const configuration = { group_id: addGroupID.value }
  const payload = { account_id: accountID, ...addDraft.value, request_model: addDraft.value.request_model.trim() }
  try {
    if (!addConfig.value || addConfig.value.user_id !== savedSettings.value?.test_user_id || addConfig.value.group_id !== configuration.group_id) addConfig.value = await identityAPI.saveConfig(accountID, configuration)
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
  addDraft.value = { request_model: row.plan.request_model, expected_model: row.plan.expected_model, interval_minutes: row.plan.interval_minutes, enabled: row.plan.enabled, options: { ...identityAPI.defaultIdentityOptions(), ...row.plan.options } }
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
async function toggleSchedule(plan: identityAPI.IdentityPlan) {
  if (loading.value || scheduleBusy.value) return
  scheduleBusy.value = plan.id
  loadError.value = ''
  try {
    await identityAPI.updatePlan(plan.id, { account_id: plan.account_id, request_model: plan.request_model, expected_model: plan.expected_model, interval_minutes: plan.interval_minutes, enabled: !plan.enabled, options: plan.options })
    await loadRows()
  } catch (error) { loadError.value = message(error) }
  finally { scheduleBusy.value = 0 }
}
async function runPlan(plan: identityAPI.IdentityPlan) {
  if (actionBusy.value || accountHasActiveRun(plan.account_id)) return
  loadError.value = ''
  actionBusy.value = plan.id
  try { await identityAPI.run(plan.id); selectedPlanID.value = plan.id; await loadRows(); await selectPlan(plan.id) }
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
  historyOpen.value = false; selectedPlanID.value = run.plan_id; reportLoading.value = true; reportError.value = ''; selectedReport.value = null
  try { const result = await identityAPI.runStatus(run.id); if (version === reportVersion) selectedReport.value = result }
  catch (error) { if (version === reportVersion) reportError.value = message(error) }
  finally { if (version === reportVersion) reportLoading.value = false }
}
async function refreshActiveReport() {
  const selected = selectedReport.value, version = reportVersion
  if (!selected || !isActive(selected)) return
  try { const result = await identityAPI.runStatus(selected.id); if (version === reportVersion && selectedReport.value?.id === selected.id) selectedReport.value = result }
  catch (error) { if (version === reportVersion) reportError.value = message(error) }
}
watch(query, () => { clearTimeout(queryTimer); queryTimer = setTimeout(() => { page.value = 1; void loadRows() }, 300) })
onMounted(async () => {
  await Promise.allSettled([loadConnectionSettings(), loadRows(), adminAPI.groups.getAll().then(result => { groups.value = result }).catch(error => { loadError.value = message(error) })])
  poll = setInterval(() => { if (!document.hidden) void refreshActiveReport(); if (!document.hidden && !loading.value && !scheduleBusy.value && !addPlanOpen.value && planRows.value.some(row => isActive(row.latest))) void loadRows() }, 2000)
})
onUnmounted(() => { accountSearchVersion++; addFormVersion++; userSearchVersion++; loadVersion++; reportVersion++; clearInterval(poll); clearTimeout(queryTimer) })
</script>
