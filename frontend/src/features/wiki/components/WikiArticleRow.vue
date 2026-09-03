<template>
  <RouterLink
    :to="wikiArticlePath(article)"
    class="group flex min-h-20 items-start gap-3 py-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500"
    :class="compact ? '' : 'sm:px-2'"
  >
    <span class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300">
      <Icon :name="articleIcon" size="sm" />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block font-semibold text-gray-950 transition-colors group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ article.title }}</span>
      <span class="mt-1 block text-sm leading-6 text-gray-600 dark:text-dark-300">{{ article.summary }}</span>
    </span>
    <Icon name="chevronRight" size="sm" class="mt-2 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5 dark:text-dark-500" />
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { wikiArticlePath, type WikiArticle } from '../content'

const props = withDefaults(defineProps<{
  article: WikiArticle
  compact?: boolean
}>(), {
  compact: false,
})

const articleIcon = computed<'terminal' | 'book' | 'exclamationCircle'>(() => {
  if (props.article.section === 'getting-started') return 'terminal'
  if (props.article.section === 'troubleshooting') return 'exclamationCircle'
  return 'book'
})
</script>
