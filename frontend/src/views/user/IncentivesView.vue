<template>
  <AppLayout><div class="mx-auto max-w-6xl space-y-6">
    <header><h1 class="text-2xl font-semibold">{{ t('incentives.title') }}</h1><p class="mt-2 text-gray-600 dark:text-gray-300">{{ t('incentives.description') }}</p></header>
    <p v-if="error" role="alert" class="text-red-600 dark:text-red-300">{{ error }} <button class="btn btn-secondary" @click="load">{{ t('incentives.refresh') }}</button></p>
    <p v-if="loading">{{ t('incentives.loading') }}</p>
    <div class="grid gap-6 lg:grid-cols-2">
      <section v-for="item in items" :id="item.kind" :key="item.kind" class="card space-y-4 p-6">
        <div class="flex flex-wrap justify-between gap-2"><h2 class="text-xl font-semibold">{{ t(`incentives.${item.kind}`) }}</h2><span>{{ t(item.enabled ? 'incentives.enabled' : 'incentives.disabled') }}</span></div>
        <p class="text-sm">{{ date(item.starts_at) }} — {{ date(item.ends_at) }} · {{ item.timezone }}</p>
        <p v-if="!item.eligible">{{ t('incentives.ineligible') }}</p>
        <template v-if="item.enabled">
          <p class="text-3xl font-semibold tabular-nums">${{ (item.kind === 'global_rate' ? item.spend : item.personal_spend).toFixed(2) }}</p>
          <progress class="h-3 w-full" :value="item.threshold - item.next_threshold" :max="item.threshold" :aria-label="t('incentives.progress')" />
          <p>{{ t('incentives.next') }} <strong>${{ item.next_threshold.toFixed(2) }}</strong></p>
          <template v-if="item.kind === 'global_rate'"><div v-for="group in item.groups" :key="group.group_id" class="flex justify-between gap-3 border-t pt-3"><span>{{ group.name }}</span><span>{{ group.base_rate.toFixed(4) }} → <strong>{{ group.current_rate.toFixed(4) }}</strong></span></div><p class="text-sm">{{ t('incentives.rateNote') }}</p></template>
          <template v-else>
            <p>{{ t('incentives.available') }}: <strong>{{ item.available }}</strong> · {{ t('incentives.earned') }}: {{ item.earned }} · {{ t('incentives.used') }}: {{ item.used }}</p>
            <p class="text-sm">{{ t('incentives.expiry') }}</p>
            <button data-testid="incentive-draw" class="btn btn-primary min-h-11" :disabled="drawing || !item.eligible || item.available < 1" @click="draw">{{ t(drawing ? 'incentives.drawing' : 'incentives.draw') }}</button>
            <p v-if="result" role="status">{{ result.prize.name }} · ${{ result.reward_amount.toFixed(2) }}</p>
            <ul class="space-y-2"><li v-for="prize in item.prizes.filter(p => p.enabled)" :key="prize.id" class="flex justify-between gap-3"><span>{{ prize.name }}</span><span>${{ prize.reward_amount }} · {{ (prize.probability * 100).toFixed(2) }}%</span></li></ul>
          </template>
        </template>
      </section>
    </div>
    <section class="card p-6"><h2 class="mb-4 text-lg font-semibold">{{ t('incentives.history') }}</h2><p v-if="!history?.rewards.length">{{ t('incentives.empty') }}</p><ul class="space-y-3"><li v-for="reward in history?.rewards" :key="reward.id" class="flex flex-wrap justify-between gap-2"><time>{{ date(reward.created_at) }}</time><span>{{ reward.prize.name }} · ${{ reward.reward_amount }}</span></li></ul></section>
  </div></AppLayout>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { incentivesAPI, type IncentiveStatus, type IncentiveHistory } from '@/api/incentives'
import type { DailyLotteryPrize } from '@/api/dailyLottery'
const { t } = useI18n()
const items = ref<IncentiveStatus[]>([]), history = ref<IncentiveHistory>(), loading = ref(false), drawing = ref(false), error = ref('')
const result = ref<{ prize: DailyLotteryPrize; reward_amount: number }>()
let requestKey: string | null = null
const date = (value: string) => new Date(value).toLocaleString()
async function load() { loading.value = true; error.value = ''; try { [items.value, history.value] = await Promise.all([incentivesAPI.status(), incentivesAPI.history()]) } catch (e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { loading.value = false } }
async function draw() { if (drawing.value) return; drawing.value = true; error.value = ''; requestKey ??= crypto.randomUUID(); try { result.value = await incentivesAPI.draw(requestKey); requestKey = null; await load() } catch (e) { error.value = e instanceof Error ? e.message : t('incentives.error') } finally { drawing.value = false } }
onMounted(load)
</script>
