<template>
  <WikiShell>
    <main class="mx-auto max-w-7xl px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
      <section class="max-w-3xl">
        <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">{{ t('wiki.eyebrow') }}</p>
        <h1 class="mt-2 text-3xl font-bold tracking-normal text-gray-950 sm:text-4xl dark:text-white">{{ t('wiki.heading') }}</h1>
        <p class="mt-4 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300">{{ t('wiki.subtitle') }}</p>

        <form class="mt-7 flex max-w-2xl gap-2" role="search" @submit.prevent="submitSearch">
          <label for="wiki-search" class="sr-only">{{ t('wiki.searchLabel') }}</label>
          <div class="relative min-w-0 flex-1">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              id="wiki-search"
              v-model="searchInput"
              type="search"
              class="h-12 w-full rounded-md border border-gray-300 bg-white pl-10 pr-4 text-base text-gray-950 outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 dark:border-dark-600 dark:bg-dark-900 dark:text-white"
              :placeholder="t('wiki.searchPlaceholder')"
              autocomplete="off"
            />
          </div>
          <button
            type="submit"
            class="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-md bg-primary-600 text-white transition-colors hover:bg-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-950"
            :title="t('wiki.searchAction')"
            :aria-label="t('wiki.searchAction')"
          >
            <Icon name="search" size="md" />
          </button>
        </form>
      </section>

      <section v-if="isSearching" class="mt-12" aria-live="polite">
        <div class="flex flex-wrap items-baseline justify-between gap-2 border-b border-gray-200 pb-4 dark:border-dark-800">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('wiki.searchResults') }}</h2>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('wiki.resultCount', { count: visibleArticles.length }) }}</p>
        </div>
        <div v-if="visibleArticles.length" class="mt-4 divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-800 dark:border-dark-800">
          <WikiArticleRow v-for="article in visibleArticles" :key="wikiArticlePath(article)" :article="article" />
        </div>
        <div v-else class="mt-6 border-l-4 border-gray-300 py-2 pl-4 dark:border-dark-700">
          <p class="font-medium text-gray-900 dark:text-white">{{ t('wiki.noResults') }}</p>
          <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('wiki.noResultsHint') }}</p>
        </div>
      </section>

      <div v-else class="mt-14 grid gap-10 lg:grid-cols-3">
        <section v-for="section in sectionsWithArticles" :key="section.id">
          <div class="border-b border-gray-200 pb-4 dark:border-dark-800">
            <h2 class="text-lg font-semibold text-gray-950 dark:text-white">{{ section.title }}</h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ section.description }}</p>
          </div>
          <div class="divide-y divide-gray-200 dark:divide-dark-800">
            <WikiArticleRow v-for="article in section.articles" :key="wikiArticlePath(article)" :article="article" compact />
          </div>
        </section>
      </div>
    </main>
  </WikiShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import WikiShell from './WikiShell.vue'
import WikiArticleRow from './components/WikiArticleRow.vue'
import { searchWikiArticles, wikiArticlePath, wikiArticles, wikiSections } from './content'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const searchInput = ref(typeof route.query.q === 'string' ? route.query.q : '')
const activeQuery = computed(() => typeof route.query.q === 'string' ? route.query.q.trim() : '')
const isSearching = computed(() => activeQuery.value.length > 0)
const visibleArticles = computed(() => searchWikiArticles(activeQuery.value))
const sectionsWithArticles = computed(() => wikiSections
  .slice()
  .sort((a, b) => a.order - b.order)
  .map((section) => ({
    ...section,
    articles: wikiArticles
      .filter((article) => article.section === section.id)
      .slice()
      .sort((a, b) => a.order - b.order),
  })))

watch(() => route.query.q, (query) => {
  searchInput.value = typeof query === 'string' ? query : ''
})

function submitSearch() {
  const query = searchInput.value.trim()
  if (!query) {
    router.push('/wiki')
    return
  }
  router.push({ name: 'WikiSearch', query: { q: query } })
}
</script>
