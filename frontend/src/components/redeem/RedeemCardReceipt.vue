<template>
  <section class="card-receipt" role="status" :aria-label="t('rewards.receiptTitle')">
    <template v-if="card">
      <div class="receipt-heading">
        <span class="receipt-kind">{{ t(`rewards.${card.template.kind}`) }}</span>
        <span class="receipt-delivered">{{ t('rewards.receiptDelivered') }}</span>
      </div>
      <h3>{{ card.template.name }}</h3>
      <p class="receipt-benefit">
        <template v-if="card.template.kind === 'model_tokens'">{{ number(card.template.tokens) }} <span>Tokens</span></template>
        <template v-else-if="card.template.kind === 'group_rate'">{{ number(card.template.rate) }}<span>× {{ t('rewards.receiptRate') }}</span></template>
        <template v-else>{{ t('rewards.receiptFree') }}</template>
      </p>
      <dl>
        <div v-if="card.template.kind !== 'group_rate'"><dt>{{ t('rewards.model') }}</dt><dd class="receipt-model">{{ card.template.model }}</dd></div>
        <div><dt>{{ t('rewards.groups') }}</dt><dd>{{ card.template.group_ids.map(id => `#${id}`).join(' · ') }}</dd></div>
        <div><dt>{{ t('rewards.receiptDuration') }}</dt><dd>{{ t('rewards.receiptHours', { hours: card.template.hours }) }}</dd></div>
        <div><dt>{{ t('rewards.claim') }}</dt><dd>{{ formatDateTime(card.claim_expires_at) }}</dd></div>
      </dl>
      <p class="receipt-note">{{ t('rewards.exchangeActivationNote') }}</p>
      <p class="receipt-note">{{ t(card.template.kind === 'group_rate' ? 'rewards.receiptRateNote' : 'rewards.receiptTextNote') }}</p>
    </template>
    <template v-else>
      <h3>{{ t('rewards.receiptDelivered') }}</h3>
      <p class="receipt-note">{{ t('rewards.notice') }}</p>
    </template>
    <RouterLink class="receipt-bag" to="/cards?tab=bag">{{ t('rewards.viewBag') }} <span aria-hidden="true">↗</span></RouterLink>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import type { RewardCard } from '@/api/rewards'
import { formatDateTime } from '@/utils/format'
defineProps<{ card?: RewardCard }>()
const { t, locale } = useI18n()
const number = (value: number) => new Intl.NumberFormat(locale?.value || 'zh-CN', { maximumFractionDigits: 8 }).format(value)
</script>

<style scoped>
.card-receipt { margin: 18px 0 22px; padding-top: 20px; border-top: 1px solid var(--rule); min-width: 0; }
.receipt-heading { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 8px 16px; font-size: 12px; line-height: 1.5; }
.receipt-kind { border: 1px solid var(--rule); border-radius: 4px; padding: 3px 8px; }
.receipt-delivered { color: theme('colors.emerald.700'); }
:global(.dark .card-receipt .receipt-delivered) { color: theme('colors.emerald.300'); }
h3 { margin-top: 16px; font-size: 20px; line-height: 1.5; font-weight: 600; overflow-wrap: anywhere; }
.receipt-benefit { margin: 8px 0 18px; font-size: clamp(24px, 5vw, 32px); line-height: 1.4; font-weight: 650; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.receipt-benefit span { margin-left: 6px; font-size: 14px; font-weight: 500; }
dl { display: grid; gap: 10px; font-size: 12px; line-height: 1.6; }
dl > div { display: grid; grid-template-columns: minmax(100px, .7fr) minmax(0, 1.3fr); gap: 12px; }
dt { color: var(--muted); }
dd { margin: 0; overflow-wrap: anywhere; }
.receipt-model { font-family: ui-monospace, monospace; }
.receipt-note { margin-top: 12px; font-size: 12px; line-height: 1.7; color: var(--muted); overflow-wrap: anywhere; }
.receipt-note + .receipt-note { margin-top: 4px; }
.receipt-bag { display: inline-flex; align-items: center; gap: 8px; min-height: 44px; margin-top: 8px; font-size: 13px; text-decoration: underline; text-underline-offset: 4px; }
.receipt-bag:focus-visible { outline: 2px solid currentColor; outline-offset: 3px; border-radius: 3px; }
@media(max-width: 550px) { dl > div { grid-template-columns: 1fr; gap: 2px; } h3 { font-size: 18px; } }
</style>
