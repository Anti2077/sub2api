<template>
  <AppLayout><div class="mx-auto max-w-6xl space-y-6">
    <header><h1 class="text-2xl font-semibold">{{ t('incentives.settings') }}</h1><p class="mt-2">{{ t('incentives.settingsNote') }}</p><p class="mt-3 rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm leading-6 text-primary-900 dark:border-primary-800 dark:bg-primary-950/30 dark:text-primary-100">{{ t('incentives.logicNote') }}</p></header>
    <p v-if="error" role="alert" class="text-red-600 dark:text-red-300">{{ error }}</p><p v-if="notice" role="status">{{ notice }}</p>
    <button class="btn btn-secondary" :disabled="loading || !!saving" @click="load()">{{ t('incentives.refresh') }}</button>
    <form v-for="config in configs" :key="config.kind" class="card space-y-5 p-6" @submit.prevent="save(config)"><fieldset :disabled="loading || !!saving" class="space-y-5">
      <h2 class="text-xl font-semibold">{{ t(`incentives.${config.kind}`) }}</h2>
      <label class="flex min-h-11 items-center gap-3"><input v-model="config.enabled" type="checkbox">{{ t('incentives.enabled') }}</label>
      <div class="grid gap-5 sm:grid-cols-2">
        <SearchMultiSelect :disabled="loading || !!saving" v-model="config.group_ids" :label="t('incentives.groups')" :options="groupOptions" :placeholder="t('incentives.selectGroups')" :search-placeholder="t('incentives.searchGroups')" :empty-text="t('incentives.noGroups')" :remove-label="t('incentives.removeOption')" />
        <SearchMultiSelect :disabled="loading || !!saving" v-model="config.excluded_user_ids" :label="t('incentives.excludedUsers')" :options="userOptions" :placeholder="t('incentives.selectUsers')" :search-placeholder="t('incentives.searchUsers')" :empty-text="t('incentives.noUsers')" :remove-label="t('incentives.removeOption')" />
        <SearchMultiSelect :disabled="loading || !!saving" v-model="config.excluded_models" :label="t('incentives.excludedModels')" :options="modelOptions" :placeholder="t('incentives.selectModels')" :search-placeholder="t('incentives.searchModels')" :empty-text="t('incentives.noModels')" :remove-label="t('incentives.removeOption')" />
        <label>{{ t('incentives.threshold') }}<input v-model.number="config.spend_threshold" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
        <template v-if="config.kind === 'global_rate'">
          <label>{{ t('incentives.decrease') }}<input v-model.number="config.rate_decrease" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
          <label>{{ t('incentives.floor') }}<input v-model.number="config.minimum_rate" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
        </template>
        <label v-else>{{ t('incentives.cap') }}<input v-model.number="config.max_chances" class="input mt-2" type="number" min="0" max="1000000" step="1" required></label>
      </div>
      <label class="flex min-h-11 items-center gap-3"><input v-model="config.exclude_admins" type="checkbox">{{ t('incentives.excludeAdmins') }}</label>
      <p>{{ t('incentives.weekly') }} · {{ config.timezone }} · {{ t('incentives.expiry') }}</p>
      <PrizeManager v-if="config.kind==='lottery'" v-model="config.prizes" :templates="cardTemplates" :groups="groups" :disabled="!!saving || loading" />
      <fieldset class="rounded-lg border p-4"><legend>{{ t('incentives.preview') }}</legend><div class="grid gap-3 sm:grid-cols-2"><label>{{ t('incentives.amount') }}<input v-model.number="previewSpend" class="input mt-2" type="number" min="0" step="any"></label><label v-if="config.kind === 'global_rate'">{{ t('incentives.baseRate') }}<input v-model.number="previewRate" class="input mt-2" type="number" min="0" step="any"></label></div><p class="mt-3 font-semibold">{{ preview(config) }}</p></fieldset>
      <button class="btn btn-primary min-h-11" :disabled="!!saving || loading">{{ t('incentives.save') }}</button></fieldset>
    </form>
    <section class="card space-y-4 p-6"><h2 class="text-xl font-semibold">{{ t('incentives.current') }}</h2><div v-for="item in statuses" :key="item.kind" class="space-y-2 border-t pt-4"><p>{{ t(`incentives.${item.kind}`) }} · ${{ item.spend.toFixed(2) }} · {{ new Date(item.ends_at).toLocaleString() }}</p><p v-for="group in item.groups" :key="group.group_id">{{ group.name }}: {{ group.base_rate }} → {{ group.current_rate }}</p><button v-if="item.period_id" class="btn btn-secondary" :disabled="loading || !!saving" @click="reset(item.period_id)">{{ t('incentives.reset') }}</button></div></section>
    <section class="space-y-5"><RecordsManager v-if="history" :items="periodRows" :title="t('rewards.managePeriodHistory')" /><RecordsManager v-if="history" :items="chanceRows" :title="t('rewards.manageChanceHistory')" /><RecordsManager v-if="history" :items="rewardRows" :title="t('rewards.manageRewardHistory')" /></section>
  </div></AppLayout>
</template>
<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import {onBeforeRouteLeave} from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PrizeManager from '@/components/rewards/PrizeManager.vue'
import {validPrizePool} from '@/components/rewards/managementValidation'
import RecordsManager from '@/components/rewards/RecordsManager.vue'
import SearchMultiSelect, { type SearchMultiSelectOption } from '@/components/common/SearchMultiSelect.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, AdminUser } from '@/types'
import { incentivesAPI, type IncentiveConfig, type IncentiveHistory, type IncentiveStatus } from '@/api/incentives'
import {rewardsAPI,type CardTemplate} from '@/api/rewards'
const cardTemplates=ref<CardTemplate[]>([])
const { t } = useI18n()
const configs = ref<IncentiveConfig[]>([]), statuses = ref<IncentiveStatus[]>([]), history = ref<IncentiveHistory>()
const loading = ref(false), saving = ref(''), error = ref(''), notice = ref(''), previewSpend = ref(100), previewRate = ref(0.45)
const periodRows = computed(() => (history.value?.periods||[]).map(v=>({...v,created_at:v.starts_at,detail:v})))
const chanceRows = computed(() => (history.value?.chances||[]).map(v=>({...v,kind:'chances',delta:v.chances})))
const rewardRows = computed(() => (history.value?.rewards||[]).map(v=>({...v,kind:v.prize.card_template_id?'card':'balance',detail:v})))
const baseline=ref('')
const dirty=()=>!!baseline.value&&JSON.stringify(configs.value)!==baseline.value
const allowDiscard=()=>!dirty()||window.confirm(t('rewards.manageDiscardDraft'))
function beforeUnload(e:BeforeUnloadEvent){if(dirty()||saving.value){e.preventDefault();e.returnValue=''}}
window.addEventListener('beforeunload',beforeUnload)
onBeforeUnmount(()=>window.removeEventListener('beforeunload',beforeUnload))
onBeforeRouteLeave(()=>!saving.value&&allowDiscard())
const groups = ref<AdminGroup[]>([]), users = ref<AdminUser[]>([]), models = ref<string[]>([])
const groupOptions = computed<SearchMultiSelectOption[]>(() => groups.value.map(group => ({ value: group.id, label: `${group.name} (#${group.id})` })))
const userOptions = computed<SearchMultiSelectOption[]>(() => users.value.map(user => ({ value: user.id, label: `${user.username || user.email} (#${user.id})${user.username && user.email ? ` · ${user.email}` : ''}` })))
const modelOptions = computed<SearchMultiSelectOption[]>(() => models.value.map(model => ({ value: model, label: model })))
async function loadSelectors() {
  const [groupResult, modelResult] = await Promise.all([adminAPI.groups.getAllIncludingInactive(), adminAPI.groups.getModelAllowlistCandidates(0)])
  groups.value = groupResult; models.value = modelResult
  const pageSize = 100; const allUsers: AdminUser[] = []
  for (let page = 1; page <= 100; page += 1) { const result = await adminAPI.users.list(page, pageSize); allUsers.push(...result.items); if (allUsers.length >= result.total || result.items.length < pageSize) break }
  users.value = allUsers
}
function preview(c: IncentiveConfig) { if (!(c.spend_threshold > 0)) return '—'; const tiers = Math.floor(Math.max(0,previewSpend.value) / c.spend_threshold); return c.kind === 'global_rate' ? Math.min(previewRate.value,Math.max(c.minimum_rate,previewRate.value - tiers*c.rate_decrease)).toFixed(4) : String(c.max_chances ? Math.min(c.max_chances,tiers) : tiers) }
async function load(force=false) { if(loading.value||(!force&&!allowDiscard()))return;loading.value = true; error.value = ''; try { [configs.value,statuses.value,history.value] = await Promise.all([incentivesAPI.configs(),incentivesAPI.status(true),incentivesAPI.history(true)]);baseline.value=JSON.stringify(configs.value) } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { loading.value = false } }
async function save(c: IncentiveConfig) { if(saving.value || loading.value)return; if(c.kind==='lottery'&&!validPrizePool(c.prizes,cardTemplates.value)){error.value=t('rewards.manageInvalidPrizePool');return} saving.value = c.kind; error.value = ''; notice.value = ''; try { const saved = await incentivesAPI.save(c); Object.assign(c,saved);const old:IncentiveConfig[]=JSON.parse(baseline.value||'[]');const index=old.findIndex(v=>v.kind===c.kind);if(index>=0)old[index]=JSON.parse(JSON.stringify(c));baseline.value=JSON.stringify(old);notice.value = t('incentives.saved'); statuses.value = await incentivesAPI.status(true) } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { saving.value = '' } }
async function reset(id: number) { if(saving.value||loading.value||!allowDiscard())return;if (!window.confirm(t('incentives.resetConfirm'))) return;saving.value='reset'; try { await incentivesAPI.reset(id); await load(true) } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally{saving.value=''} }
onMounted(async () => { try { await Promise.all([load(), loadSelectors(),rewardsAPI.templates().then(v=>{cardTemplates.value=v})]) } catch (e) { error.value = e instanceof Error ? e.message : t('incentives.error') } })
</script>
