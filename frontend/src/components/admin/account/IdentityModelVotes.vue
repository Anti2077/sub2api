<template>
  <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="identity-model-votes" :aria-label="t('admin.modelIdentity.modelVotes')">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h3 class="text-sm font-semibold">{{ t('admin.modelIdentity.modelVotes') }}</h3>
      <span v-if="votes.length" class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.modelVotesTotal', { count: total }) }}</span>
    </div>
    <dl v-if="votes.length" class="mt-3 space-y-2">
      <div v-for="entry in votes" :key="entry.model" class="flex items-start justify-between gap-3 rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-900">
        <dt class="min-w-0 break-all text-sm">{{ entry.model }}</dt>
        <dd class="shrink-0 rounded-md bg-gray-100 px-2 py-0.5 text-sm font-semibold tabular-nums text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t('admin.modelIdentity.modelVoteCount', { count: entry.votes }) }}</dd>
      </div>
    </dl>
    <p v-else class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.modelVotesUnavailable') }}</p>
    <p v-if="votes.length" class="mt-3 text-xs leading-relaxed text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.modelVotesHint') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { identityModelVotes } from '@/utils/modelIdentity'

const props = defineProps<{ report?: Record<string, unknown> }>()
const { t } = useI18n()
const votes = computed(() => identityModelVotes(props.report))
const total = computed(() => votes.value.reduce((sum, entry) => sum + entry.votes, 0))
</script>
