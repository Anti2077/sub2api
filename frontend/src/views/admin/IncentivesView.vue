<template>
  <AppLayout><div class="mx-auto max-w-6xl space-y-6">
    <header><h1 class="text-2xl font-semibold">{{ t('incentives.settings') }}</h1><p class="mt-2">{{ t('incentives.settingsNote') }}</p></header>
    <p v-if="error" role="alert" class="text-red-600 dark:text-red-300">{{ error }}</p><p v-if="notice" role="status">{{ notice }}</p>
    <button class="btn btn-secondary" :disabled="loading" @click="load">{{ t('incentives.refresh') }}</button>
    <form v-for="config in configs" :key="config.kind" class="card space-y-5 p-6" @submit.prevent="save(config)">
      <h2 class="text-xl font-semibold">{{ t(`incentives.${config.kind}`) }}</h2>
      <label class="flex min-h-11 items-center gap-3"><input v-model="config.enabled" type="checkbox">{{ t('incentives.enabled') }}</label>
      <div class="grid gap-5 sm:grid-cols-2">
        <label>{{ t('incentives.groups') }}<input class="input mt-2" :value="config.group_ids.join(', ')" @change="config.group_ids = ids($event)"><small>{{ t('incentives.idsHint') }}</small></label>
        <label>{{ t('incentives.excludedUsers') }}<input class="input mt-2" :value="config.excluded_user_ids.join(', ')" @change="config.excluded_user_ids = ids($event)"></label>
        <label>{{ t('incentives.excludedModels') }}<input class="input mt-2" :value="config.excluded_models.join(', ')" @change="config.excluded_models = strings($event)"></label>
        <label>{{ t('incentives.threshold') }}<input v-model.number="config.spend_threshold" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
        <template v-if="config.kind === 'global_rate'">
          <label>{{ t('incentives.decrease') }}<input v-model.number="config.rate_decrease" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
          <label>{{ t('incentives.floor') }}<input v-model.number="config.minimum_rate" class="input mt-2" type="number" min="0.0000000001" step="any" required></label>
        </template>
        <label v-else>{{ t('incentives.cap') }}<input v-model.number="config.max_chances" class="input mt-2" type="number" min="0" max="1000000" step="1" required></label>
      </div>
      <label class="flex min-h-11 items-center gap-3"><input v-model="config.exclude_admins" type="checkbox">{{ t('incentives.excludeAdmins') }}</label>
      <p>{{ t('incentives.weekly') }} · {{ config.timezone }} · {{ t('incentives.expiry') }}</p>
      <fieldset v-if="config.kind === 'lottery'" class="space-y-4"><legend class="font-semibold">{{ t('incentives.prizes') }}</legend>
        <div v-for="(prize, index) in config.prizes" :key="prize.id" class="grid gap-3 rounded-lg border p-4 sm:grid-cols-4">
          <label>{{ t('incentives.prizeName') }}<input v-model="prize.name" class="input mt-1" required maxlength="40"></label>
          <label>{{ t('incentives.amount') }}<input v-model.number="prize.reward_amount" class="input mt-1" type="number" min="0" max="1000000" step="0.00000001" required></label>
          <label>{{ t('incentives.weight') }}<input v-model.number="prize.weight" class="input mt-1" type="number" min="1" max="1000000000" step="1" required></label>
          <div><label class="flex min-h-11 items-center gap-2"><input v-model="prize.enabled" type="checkbox">{{ t('incentives.enabled') }}</label><button type="button" class="btn btn-secondary" :disabled="config.prizes.length <= 2" @click="config.prizes.splice(index,1)">{{ t('incentives.remove') }}</button></div>
        </div>
        <button type="button" class="btn btn-secondary" :disabled="config.prizes.length >= 8" @click="config.prizes.push({id: cryptoID(),name:'',reward_amount:0,weight:1,enabled:true})">{{ t('incentives.addPrize') }}</button>
      </fieldset>
      <fieldset class="rounded-lg border p-4"><legend>{{ t('incentives.preview') }}</legend><div class="grid gap-3 sm:grid-cols-2"><label>{{ t('incentives.amount') }}<input v-model.number="previewSpend" class="input mt-2" type="number" min="0" step="any"></label><label v-if="config.kind === 'global_rate'">{{ t('incentives.baseRate') }}<input v-model.number="previewRate" class="input mt-2" type="number" min="0" step="any"></label></div><p class="mt-3 font-semibold">{{ preview(config) }}</p></fieldset>
      <button class="btn btn-primary min-h-11" :disabled="saving === config.kind">{{ t('incentives.save') }}</button>
    </form>
    <section class="card space-y-4 p-6"><h2 class="text-xl font-semibold">{{ t('incentives.current') }}</h2><div v-for="item in statuses" :key="item.kind" class="space-y-2 border-t pt-4"><p>{{ t(`incentives.${item.kind}`) }} · ${{ item.spend.toFixed(2) }} · {{ new Date(item.ends_at).toLocaleString() }}</p><p v-for="group in item.groups" :key="group.group_id">{{ group.name }}: {{ group.base_rate }} → {{ group.current_rate }}</p><button v-if="item.period_id" class="btn btn-secondary" @click="reset(item.period_id)">{{ t('incentives.reset') }}</button></div></section>
    <section class="card p-6"><h2 class="mb-4 text-xl font-semibold">{{ t('incentives.history') }}</h2>
      <div v-for="period in history?.periods" :key="period.id" class="flex flex-wrap justify-between gap-3 border-t py-3"><span>#{{ period.id }} · {{ period.kind }}</span><span>{{ new Date(period.starts_at).toLocaleString() }}</span><span>${{ period.spend }}</span></div>
      <h3 class="my-4 font-semibold">{{ t('incentives.earned') }}</h3><p v-for="chance in history?.chances" :key="chance.id">#{{ chance.user_id }} · +{{ chance.chances }} · {{ new Date(chance.created_at).toLocaleString() }}</p>
      <h3 class="my-4 font-semibold">{{ t('incentives.prizes') }}</h3><p v-for="reward in history?.rewards" :key="reward.id">#{{ reward.user_id }} · {{ reward.prize.name }} · ${{ reward.reward_amount }}</p>
    </section>
  </div></AppLayout>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { incentivesAPI, type IncentiveConfig, type IncentiveHistory, type IncentiveStatus } from '@/api/incentives'
const { t } = useI18n()
const configs = ref<IncentiveConfig[]>([]), statuses = ref<IncentiveStatus[]>([]), history = ref<IncentiveHistory>()
const loading = ref(false), saving = ref(''), error = ref(''), notice = ref(''), previewSpend = ref(100), previewRate = ref(0.45)
const cryptoID = () => crypto.randomUUID()
const strings = (e: Event) => (e.target as HTMLInputElement).value.split(/[,，\n]/).map(v => v.trim()).filter(Boolean)
const ids = (e: Event) => strings(e).map(Number)
function preview(c: IncentiveConfig) { if (!(c.spend_threshold > 0)) return '—'; const tiers = Math.floor(Math.max(0,previewSpend.value) / c.spend_threshold); return c.kind === 'global_rate' ? Math.min(previewRate.value,Math.max(c.minimum_rate,previewRate.value - tiers*c.rate_decrease)).toFixed(4) : String(c.max_chances ? Math.min(c.max_chances,tiers) : tiers) }
async function load() { loading.value = true; error.value = ''; try { [configs.value,statuses.value,history.value] = await Promise.all([incentivesAPI.configs(),incentivesAPI.status(true),incentivesAPI.history(true)]) } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { loading.value = false } }
async function save(c: IncentiveConfig) { saving.value = c.kind; error.value = ''; notice.value = ''; try { const saved = await incentivesAPI.save(c); Object.assign(c,saved); notice.value = t('incentives.saved'); statuses.value = await incentivesAPI.status(true) } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { saving.value = '' } }
async function reset(id: number) { if (!window.confirm(t('incentives.resetConfirm'))) return; try { await incentivesAPI.reset(id); await load() } catch(e) { error.value = e instanceof Error ? e.message : t('incentives.error') } }
onMounted(load)
</script>
