<template>
  <button
    type="button"
    role="switch"
    :aria-checked="enabled"
    :aria-busy="busy"
    :aria-label="t('admin.modelIdentity.scheduleToggleLabel', { id: planId })"
    :disabled="disabled || busy"
    class="flex min-h-11 flex-wrap items-center gap-x-2 rounded text-left text-xs focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 disabled:cursor-not-allowed disabled:opacity-50"
    @click="$emit('change')"
  >
    <span class="inline-flex h-6 w-11 shrink-0 items-center rounded-full p-0.5 transition-colors" :class="enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'" aria-hidden="true">
      <span class="h-5 w-5 rounded-full bg-white shadow transition-transform motion-reduce:transition-none" :class="enabled ? 'translate-x-5' : 'translate-x-0'" />
    </span>
    <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ busy ? t('common.saving') : t(enabled ? 'admin.modelIdentity.scheduleOn' : 'admin.modelIdentity.scheduleOff') }}</span>
  </button>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
defineProps<{ planId: number; enabled: boolean; busy: boolean; disabled: boolean }>()
defineEmits<{ change: [] }>()
const { t } = useI18n()
</script>
