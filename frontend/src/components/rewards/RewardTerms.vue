<template>
  <dl class="reward-terms">
    <div v-if="template.kind !== 'group_rate'"><dt>{{ t('rewards.model') }}</dt><dd>{{ templateModels(template).join(', ') }}</dd></div>
    <div><dt>{{ t('rewards.groups') }}</dt><dd>{{ template.group_ids.map(id => `#${id}`).join(', ') }}</dd></div>
    <div v-if="template.kind === 'group_rate'"><dt>{{ t('rewards.rate') }}</dt><dd>{{ template.rate }}×</dd></div>
    <div v-if="template.kind === 'model_tokens'"><dt>{{ t('rewards.tokens') }}</dt><dd>{{ (remainingTokens ?? template.tokens).toLocaleString() }} Token</dd></div>
    <div><dt>{{ t('rewards.hoursLabel') }}</dt><dd>{{ template.hours }} {{ t('rewards.hours') }}</dd></div>
  </dl>
</template>

<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import { useI18n } from 'vue-i18n'
import type { CardTemplate } from '@/api/rewards'
defineProps<{ template: CardTemplate; remainingTokens?: number }>()
const { t } = useI18n()
</script>
