<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('profile.confirmUsernameTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('profile.confirmUsernameDescription') }}
        </p>
      </div>

      <form class="space-y-5" @submit.prevent="submit">
        <div>
          <label for="username-confirm" class="input-label">{{ t('profile.username') }}</label>
          <input
            id="username-confirm"
            v-model="username"
            type="text"
            required
            autofocus
            autocomplete="username"
            maxlength="100"
            class="input"
            :placeholder="t('profile.enterUsername')"
          />
          <p class="input-hint">{{ t('profile.confirmUsernameHint') }}</p>
        </div>

        <button type="submit" class="btn btn-primary w-full" :disabled="saving">
          {{ saving ? t('common.saving') : t('profile.confirmUsernameAction') }}
        </button>
      </form>

      <button type="button" class="btn btn-secondary w-full" @click="logout">
        {{ t('nav.logout') }}
      </button>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { userAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import type { UserAuthProvider, UserAuthBindingStatus } from '@/types'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const providerOrder: UserAuthProvider[] = ['github', 'google', 'oidc', 'linuxdo', 'wechat', 'dingtalk']
const identitySuggestions = computed(() => authStore.user?.identities || {})
const suggestedUsername = computed(() => {
  for (const provider of providerOrder) {
    const identity = identitySuggestions.value[provider] as UserAuthBindingStatus | undefined
    const candidate = identity?.display_name?.trim() || ''
    if (candidate && !candidate.includes('@')) {
      return candidate
    }
  }
  return ''
})
const username = ref(authStore.user?.username || suggestedUsername.value)
const saving = ref(false)

function safeRedirect(): string {
  const value = typeof route.query.redirect === 'string' ? route.query.redirect : ''
  if (value.startsWith('/') && !value.startsWith('//') && !value.startsWith('/username-confirm')) {
    return value
  }
  return authStore.isAdmin ? '/admin/dashboard' : '/dashboard'
}

async function submit(): Promise<void> {
  const normalized = username.value.trim()
  if (!normalized) {
    appStore.showError(t('profile.usernameRequired'))
    return
  }
  if (containsControlCharacters(normalized) || [...normalized].length > 100) {
    appStore.showError(t('profile.usernameInvalid'))
    return
  }

  saving.value = true
  try {
    const updated = await userAPI.updateProfile({ username: normalized })
    authStore.user = updated
    appStore.showSuccess(t('profile.updateSuccess'))
    await router.replace(safeRedirect())
  } catch (error: unknown) {
    const message = extractApiErrorCode(error) === 'USERNAME_EXISTS'
      ? t('profile.usernameExists')
      : extractApiErrorMessage(error) || t('profile.updateFailed')
    appStore.showError(message)
  } finally {
    saving.value = false
  }
}

function containsControlCharacters(value: string): boolean {
  return [...value].some((character) => {
    const codePoint = character.codePointAt(0) ?? 0
    return codePoint <= 0x1f || (codePoint >= 0x7f && codePoint <= 0x9f)
  })
}

async function logout(): Promise<void> {
  await authStore.logout()
  await router.replace('/login')
}
</script>
