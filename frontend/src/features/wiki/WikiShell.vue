<template>
  <div class="min-h-screen bg-canvas-100 text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <header class="sticky top-0 z-30 border-b border-gray-200 bg-white/95 dark:border-dark-800 dark:bg-dark-950/95">
      <nav class="mx-auto flex min-h-16 max-w-7xl items-center gap-3 px-4 sm:px-6 lg:px-8" :aria-label="t('wiki.primaryNavigation')">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-3 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500">
          <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="h-9 w-9 shrink-0 rounded-md object-contain" />
          <span class="hidden min-w-0 truncate text-sm font-semibold sm:block">{{ siteName }}</span>
        </RouterLink>

        <span class="hidden h-5 w-px bg-gray-200 sm:block dark:bg-dark-700"></span>
        <RouterLink to="/wiki" class="shrink-0 text-sm font-semibold text-primary-700 dark:text-primary-300">
          {{ t('wiki.title') }}
        </RouterLink>

        <div class="ml-auto flex items-center gap-1 sm:gap-2">
          <RouterLink
            to="/home"
            class="inline-flex min-h-11 items-center gap-2 rounded-md px-3 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
          >
            <Icon name="home" size="sm" />
            <span class="hidden sm:inline">{{ t('wiki.backHome') }}</span>
          </RouterLink>
          <button
            type="button"
            class="inline-flex h-11 w-11 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="themeLabel"
            :aria-label="themeLabel"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <RouterLink
            :to="accountPath"
            class="inline-flex min-h-11 items-center justify-center rounded-md bg-gray-900 px-3 text-sm font-semibold text-white transition-colors hover:bg-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200 dark:focus-visible:ring-offset-dark-950"
          >
            {{ accountLabel }}
          </RouterLink>
        </div>
      </nav>
    </header>

    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const isDark = ref(document.documentElement.classList.contains('dark'))

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const accountPath = computed(() => {
  if (!authStore.isAuthenticated) return '/login'
  return authStore.isAdmin ? '/admin/dashboard' : '/dashboard'
})
const accountLabel = computed(() => authStore.isAuthenticated ? t('wiki.dashboard') : t('wiki.login'))
const themeLabel = computed(() => isDark.value ? t('nav.lightMode') : t('nav.darkMode'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
