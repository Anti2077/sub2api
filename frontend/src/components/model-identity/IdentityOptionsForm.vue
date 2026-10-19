<template>
  <details class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
    <summary class="flex min-h-11 cursor-pointer items-center text-sm font-medium">{{ t('admin.modelIdentity.advancedOptions') }}</summary>
    <div class="mt-3 grid gap-4 sm:grid-cols-2">
      <label class="text-sm">{{ t('admin.modelIdentity.protocol') }}<select v-model="options.protocol" class="input mt-1 min-h-11 w-full"><option value="chat_completions">Chat Completions</option><option value="responses">Responses</option><option value="messages">Messages</option></select></label>
      <label class="text-sm">{{ t('admin.modelIdentity.reasoningEffort') }}<select v-model="options.reasoning_effort" class="input mt-1 min-h-11 w-full"><option value="">{{ t('admin.modelIdentity.defaultOption') }}</option><option v-for="value in ['none','minimal','low','medium','high','xhigh','max']" :key="value" :value="value">{{ value }}</option></select></label>
      <label class="text-sm">{{ t('admin.modelIdentity.serviceTier') }}<select v-model="options.service_tier" class="input mt-1 min-h-11 w-full"><option value="">{{ t('admin.modelIdentity.defaultOption') }}</option><option v-for="value in ['flex','fast','ultrafast']" :key="value" :value="value">{{ value }}</option></select></label>
      <div class="grid grid-cols-2 gap-2 text-sm"><label class="flex min-h-11 items-center gap-2"><input v-model="options.stream" type="checkbox" />{{ t('admin.modelIdentity.streaming') }}</label><label class="flex min-h-11 items-center gap-2"><input v-model="options.parallel" type="checkbox" />{{ t('admin.modelIdentity.parallel') }}</label><label class="flex min-h-11 items-center gap-2"><input v-model="options.relaxed" type="checkbox" />{{ t('admin.modelIdentity.relaxed') }}</label><label class="flex min-h-11 items-center gap-2"><input v-model="options.tokenizer" type="checkbox" />{{ t('admin.modelIdentity.tokenizer') }}</label></div>
    </div>
    <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.optionsHint') }}</p>
    <p v-if="options.tokenizer" role="status" class="mt-3 rounded-lg bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">{{ t('admin.modelIdentity.tokenizerCostHint') }}</p>
  </details>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { IdentityOptions } from '@/api/admin/modelIdentity'
const options = defineModel<IdentityOptions>({ required: true })
const { t } = useI18n()
</script>
