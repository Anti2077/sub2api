<template>
  <div :class="props.embedded ? 'space-y-4' : 'card'">
    <div
      v-if="!props.embedded"
      class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
    >
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.editProfile') }}
      </h2>
    </div>
    <div :class="props.embedded ? '' : 'px-6 py-6'">
      <form @submit.prevent="handleUpdateProfile" class="space-y-4">
        <div v-if="props.embedded">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('profile.editProfile') }}
          </p>
        </div>
        <div>
          <label for="username" class="input-label">
            {{ t('profile.username') }}
          </label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="input"
            :placeholder="t('profile.enterUsername')"
          />
        </div>

        <div class="flex items-center justify-between gap-4 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('profile.leaderboardAnonymous') }}</p>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('profile.leaderboardAnonymousHint') }}</p>
          </div>
          <Toggle v-model="leaderboardAnonymous" :disabled="!usernameConfirmed" />
        </div>

        <div class="flex justify-end pt-4">
          <button type="submit" :disabled="loading" class="btn btn-primary">
            {{ loading ? t('profile.updating') : t('profile.updateProfile') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userAPI } from '@/api'
import Toggle from '@/components/common/Toggle.vue'

const props = withDefaults(defineProps<{
  initialUsername: string
  initialLeaderboardAnonymous?: boolean
  usernameConfirmed?: boolean
  embedded?: boolean
}>(), {
  embedded: false,
})

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const username = ref(props.initialUsername)
const leaderboardAnonymous = ref(props.initialLeaderboardAnonymous ?? false)
const usernameConfirmed = props.usernameConfirmed ?? Boolean(props.initialUsername.trim())
const loading = ref(false)

watch(() => props.initialUsername, (val) => {
  username.value = val
})

watch(() => props.initialLeaderboardAnonymous, (val) => {
  leaderboardAnonymous.value = val ?? false
})

const handleUpdateProfile = async () => {
  if (!username.value.trim()) {
    appStore.showError(t('profile.usernameRequired'))
    return
  }

  loading.value = true
  try {
    const updatedUser = await userAPI.updateProfile({
      username: username.value,
      leaderboard_anonymous: leaderboardAnonymous.value
    })
    authStore.user = updatedUser
    appStore.showSuccess(t('profile.updateSuccess'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('profile.updateFailed'))
  } finally {
    loading.value = false
  }
}
</script>
