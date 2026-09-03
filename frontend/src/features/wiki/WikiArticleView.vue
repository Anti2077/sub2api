<template>
  <WikiShell>
    <main class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div v-if="article" class="grid gap-10 lg:grid-cols-[220px_minmax(0,1fr)_200px]">
        <aside class="hidden lg:block">
          <nav class="sticky top-24" :aria-label="t('wiki.articleNavigation')">
            <RouterLink to="/wiki" class="mb-5 inline-flex items-center gap-2 text-sm font-medium text-gray-600 hover:text-primary-700 dark:text-dark-300 dark:hover:text-primary-300">
              <Icon name="arrowLeft" size="sm" />
              {{ t('wiki.allArticles') }}
            </RouterLink>
            <div v-for="section in sectionsWithArticles" :key="section.id" class="mb-6">
              <p class="mb-2 text-xs font-semibold uppercase text-gray-500 dark:text-dark-400">{{ section.title }}</p>
              <RouterLink
                v-for="item in section.articles"
                :key="wikiArticlePath(item)"
                :to="wikiArticlePath(item)"
                class="block border-l-2 py-1.5 pl-3 text-sm leading-5 transition-colors"
                :class="item.slug === article.slug && item.section === article.section
                  ? 'border-primary-600 font-semibold text-primary-700 dark:text-primary-300'
                  : 'border-gray-200 text-gray-600 hover:border-gray-400 hover:text-gray-950 dark:border-dark-700 dark:text-dark-300 dark:hover:border-dark-500 dark:hover:text-white'"
              >
                {{ item.title }}
              </RouterLink>
            </div>
          </nav>
        </aside>

        <article class="min-w-0">
          <nav class="mb-6 flex flex-wrap items-center gap-2 text-sm text-gray-500 dark:text-dark-400" :aria-label="t('wiki.breadcrumbs')">
            <RouterLink to="/wiki" class="hover:text-primary-700 dark:hover:text-primary-300">{{ t('wiki.title') }}</RouterLink>
            <Icon name="chevronRight" size="xs" />
            <span>{{ currentSection?.title }}</span>
          </nav>

          <header class="border-b border-gray-200 pb-6 dark:border-dark-800">
            <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">{{ currentSection?.title }}</p>
            <h1 class="mt-2 text-3xl font-bold tracking-normal text-gray-950 sm:text-4xl dark:text-white">{{ article.title }}</h1>
            <p class="mt-4 text-base leading-7 text-gray-600 dark:text-dark-300">{{ article.summary }}</p>
            <p v-if="article.lastVerified" class="mt-4 text-sm text-gray-500 dark:text-dark-400">
              {{ t('wiki.lastVerified', { date: article.lastVerified }) }}
              <span v-if="article.verifiedWith"> · {{ article.verifiedWith }}</span>
            </p>
          </header>

          <div
            ref="contentElement"
            class="wiki-article-content"
            v-html="rendered.html"
            @click="handleContentClick"
          ></div>
        </article>

        <aside v-if="rendered.headings.length" class="hidden xl:block">
          <nav class="sticky top-24" :aria-label="t('wiki.tableOfContents')">
            <p class="mb-3 text-xs font-semibold uppercase text-gray-500 dark:text-dark-400">{{ t('wiki.onThisPage') }}</p>
            <a
              v-for="heading in rendered.headings"
              :key="heading.id"
              :href="`#${heading.id}`"
              class="block py-1.5 text-sm leading-5 text-gray-600 hover:text-primary-700 dark:text-dark-300 dark:hover:text-primary-300"
              :class="heading.level === 3 ? 'pl-3' : ''"
            >
              {{ heading.text }}
            </a>
          </nav>
        </aside>
      </div>

      <section v-else class="mx-auto max-w-2xl py-20 text-center">
        <Icon name="document" size="xl" class="mx-auto text-gray-400 dark:text-dark-500" />
        <h1 class="mt-5 text-2xl font-bold text-gray-950 dark:text-white">{{ t('wiki.articleNotFound') }}</h1>
        <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('wiki.articleNotFoundHint') }}</p>
        <RouterLink to="/wiki" class="mt-7 inline-flex min-h-11 items-center gap-2 rounded-md bg-primary-600 px-4 text-sm font-semibold text-white hover:bg-primary-700">
          <Icon name="arrowLeft" size="sm" />
          {{ t('wiki.allArticles') }}
        </RouterLink>
      </section>
    </main>
  </WikiShell>
</template>

<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import WikiShell from './WikiShell.vue'
import { getWikiArticle, wikiArticlePath, wikiArticles, wikiSections } from './content'
import { renderWikiMarkdown } from './markdown'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const article = computed(() => getWikiArticle(String(route.params.section || ''), String(route.params.slug || '')))
const currentSection = computed(() => wikiSections.find((section) => section.id === article.value?.section))
const rendered = computed(() => renderWikiMarkdown(article.value?.source || ''))
const sectionsWithArticles = computed(() => wikiSections
  .slice()
  .sort((a, b) => a.order - b.order)
  .map((section) => ({
    ...section,
    articles: wikiArticles
      .filter((item) => item.section === section.id)
      .slice()
      .sort((a, b) => a.order - b.order),
  })))

watchEffect(() => {
  if (article.value) {
    document.title = `${article.value.title} - ${appStore.siteName || 'Sub2API'}`
  }
})

function handleContentClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const link = target?.closest<HTMLAnchorElement>('a[href]')
  const href = link?.getAttribute('href') || ''
  if (!href.startsWith('/wiki/')) return
  event.preventDefault()
  router.push(href)
}
</script>

<style scoped>
.wiki-article-content {
  @apply pt-7 text-base leading-7 text-gray-700 dark:text-dark-200;
  overflow-wrap: anywhere;
}

.wiki-article-content :deep(h1) {
  display: none;
}

.wiki-article-content :deep(h2) {
  @apply mb-3 mt-10 scroll-mt-24 text-2xl font-bold tracking-normal text-gray-950 dark:text-white;
}

.wiki-article-content :deep(h3) {
  @apply mb-2 mt-8 scroll-mt-24 text-xl font-semibold text-gray-950 dark:text-white;
}

.wiki-article-content :deep(p) {
  @apply mb-5;
}

.wiki-article-content :deep(ul),
.wiki-article-content :deep(ol) {
  @apply mb-5 space-y-2 pl-6;
}

.wiki-article-content :deep(ul) {
  @apply list-disc;
}

.wiki-article-content :deep(ol) {
  @apply list-decimal;
}

.wiki-article-content :deep(a) {
  @apply font-medium text-primary-700 underline underline-offset-4 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200;
}

.wiki-article-content :deep(blockquote) {
  @apply my-6 border-l-4 border-amber-400 bg-amber-50 px-4 py-3 text-amber-950 dark:border-amber-500 dark:bg-amber-500/10 dark:text-amber-100;
}

.wiki-article-content :deep(blockquote p:last-child) {
  @apply mb-0;
}

.wiki-article-content :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-sm text-gray-900 dark:bg-dark-800 dark:text-gray-100;
}

.wiki-article-content :deep(pre) {
  @apply my-6 overflow-x-auto rounded-md border border-gray-800 bg-gray-950 p-4 text-sm leading-6 text-gray-100;
}

.wiki-article-content :deep(pre code) {
  @apply bg-transparent p-0 text-inherit;
}

.wiki-article-content :deep(table) {
  @apply my-6 block w-full overflow-x-auto border-collapse text-sm;
}

.wiki-article-content :deep(th),
.wiki-article-content :deep(td) {
  @apply border border-gray-300 px-3 py-2 text-left align-top dark:border-dark-600;
}

.wiki-article-content :deep(th) {
  @apply bg-gray-100 font-semibold text-gray-950 dark:bg-dark-800 dark:text-white;
}

.wiki-article-content :deep(hr) {
  @apply my-8 border-gray-200 dark:border-dark-800;
}
</style>
