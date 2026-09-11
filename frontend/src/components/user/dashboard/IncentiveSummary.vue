<template>
  <section class="card p-5" aria-labelledby="incentive-summary-title">
    <RouterLink id="incentive-summary-title" to="/incentives" class="font-semibold text-primary-600 dark:text-primary-300">{{ t('incentives.title') }} →</RouterLink>
    <div v-if="items.length" class="mt-3 flex flex-wrap gap-4 text-sm">
      <template v-for="item in items" :key="item.kind">
        <span v-if="!item.enabled">{{ t(`incentives.${item.kind}`) }} · {{ t('incentives.disabled') }}</span>
        <template v-else-if="item.kind === 'global_rate'">
          <span v-for="group in item.groups" :key="group.group_id">{{ group.name }} · {{ group.current_rate.toFixed(4) }}</span>
          <span>{{ t('incentives.next') }} ${{ item.next_threshold.toFixed(2) }}</span>
        </template>
        <span v-else>{{ t('incentives.available') }}: {{ item.available }}</span>
      </template>
    </div>
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { incentivesAPI, type IncentiveStatus } from '@/api/incentives'
const { t } = useI18n()
const items = ref<IncentiveStatus[]>([])
onMounted(async () => { try { items.value = await incentivesAPI.status() } catch { /* Link remains available when the summary cannot load. */ } })
</script>
