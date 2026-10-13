<template>
  <div v-if="identityFlagshipClass(run)" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200" data-testid="identity-scope-summary">
    <p class="font-medium">{{ t('admin.modelIdentity.flagshipRange') }}</p>
    <p class="mt-1">{{ t('admin.modelIdentity.flagshipHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { IdentityRun } from '@/api/admin/modelIdentity'
import { identityFlagshipClass } from '@/utils/modelIdentity'

defineProps<{ run?: IdentityRun | null }>()
const { t } = useI18n()
</script>
