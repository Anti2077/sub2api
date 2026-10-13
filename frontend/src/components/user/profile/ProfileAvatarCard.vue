<template>
  <div :class="props.embedded ? 'space-y-4' : 'card'">
    <div
      v-if="!props.embedded"
      class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
    >
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.avatar.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.avatar.description') }}
      </p>
    </div>

    <div :class="props.embedded ? 'space-y-3' : 'flex flex-col gap-5 px-6 py-6 sm:flex-row sm:items-start'">
      <div
        :class="props.embedded
          ? 'flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 text-xl font-bold text-white shadow-lg shadow-primary-500/20'
          : 'flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 text-3xl font-bold text-white shadow-lg shadow-primary-500/20'"
      >
        <img
          v-if="avatarPreviewUrl"
          data-testid="profile-avatar-preview"
          :src="avatarPreviewUrl"
          :alt="displayName"
          class="h-full w-full object-cover"
        >
        <span v-else>{{ avatarInitial }}</span>
      </div>

      <div :class="props.embedded ? 'space-y-3' : 'min-w-0 flex-1 space-y-4'">
        <div class="space-y-1">
          <p v-if="props.embedded" class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('profile.avatar.title') }}
          </p>
          <p v-else class="text-sm font-medium text-gray-900 dark:text-white">
            {{ displayName }}
          </p>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('profile.avatar.uploadHint') }}
          </p>
        </div>

        <p v-if="avatarReading || avatarSaving" role="status" class="text-sm text-gray-500 dark:text-gray-400">
          {{ t(avatarReading ? 'profile.avatar.reading' : 'profile.avatar.saving') }}
        </p>

        <div class="flex flex-wrap items-center gap-3" :aria-busy="avatarReading || avatarSaving">
          <label class="btn btn-secondary btn-sm cursor-pointer">
            <input
              data-testid="profile-avatar-file-input"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              :disabled="avatarSaving || avatarReading"
              class="hidden"
              @change="handleAvatarFileChange"
            >
            {{ t('profile.avatar.uploadAction') }}
          </label>

          <button
            data-testid="profile-avatar-save"
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="avatarSaving || avatarReading || !avatarDraft"
            @click="handleAvatarSave"
          >
            {{ avatarSaving ? t('profile.avatar.saving') : t('common.save') }}
          </button>

          <button
            data-testid="profile-avatar-delete"
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="avatarSaving || avatarReading"
            @click="handleAvatarDelete"
          >
            {{ t('common.delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = withDefaults(defineProps<{
  user: User | null
  embedded?: boolean
}>(), {
  embedded: false,
})

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const maxAvatarUploadBytes = 10 * 1024 * 1024
const supportedAvatarTypes = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])
const avatarDraft = ref('')
const avatarSaving = ref(false)
const avatarReading = ref(false)

const displayName = computed(() => props.user?.username?.trim() || props.user?.email?.trim() || t('profile.user'))
const avatarInitial = computed(() => displayName.value.charAt(0).toUpperCase() || 'U')
const avatarPreviewUrl = computed(() => avatarDraft.value.trim() || props.user?.avatar_url?.trim() || '')

watch(
  () => props.user?.avatar_url,
  () => {
    avatarDraft.value = ''
  }
)

function normalizeUploadedAvatar(value: string): string | null {
  const normalized = value.trim()
  if (!normalized) {
    return null
  }

  if (!/^data:image\/[a-zA-Z0-9.+-]+;base64,/i.test(normalized)) {
    appStore.showError(t('profile.avatar.uploadRequired'))
    return null
  }

  return normalized
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error ?? new Error('avatar_read_failed'))
    reader.readAsDataURL(file)
  })
}

async function handleAvatarFileChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0]
  if (input) {
    input.value = ''
  }
  if (!file || avatarSaving.value || avatarReading.value) {
    return
  }

  avatarDraft.value = ''
  avatarReading.value = true
  try {
    if (!supportedAvatarTypes.has(file.type)) {
      throw new Error(t('profile.avatar.invalidType'))
    }
    if (file.size > maxAvatarUploadBytes) {
      throw new Error(t('profile.avatar.uploadTooLarge'))
    }
    // Send the original to the server so GIF frames survive compression.
    const dataURL = await readFileAsDataURL(file)
    const normalized = normalizeUploadedAvatar(dataURL)
    if (!normalized) {
      return
    }
    avatarDraft.value = normalized
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    avatarReading.value = false
  }
}

async function handleAvatarSave() {
  if (avatarSaving.value || avatarReading.value) return
  const normalized = normalizeUploadedAvatar(avatarDraft.value)
  if (!normalized) {
    return
  }

  avatarSaving.value = true
  try {
    const updated = await userAPI.updateProfile({ avatar_url: normalized })
    authStore.user = updated
    avatarDraft.value = ''
    appStore.showSuccess(t('profile.avatar.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    avatarSaving.value = false
  }
}

async function handleAvatarDelete() {
  if (avatarSaving.value || avatarReading.value) {
    return
  }
  if (!avatarDraft.value.trim() && !props.user?.avatar_url?.trim()) {
    appStore.showError(t('profile.avatar.emptyDeleteHint'))
    return
  }

  avatarSaving.value = true
  try {
    const updated = await userAPI.updateProfile({ avatar_url: '' })
    authStore.user = updated
    avatarDraft.value = ''
    appStore.showSuccess(t('profile.avatar.deleteSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    avatarSaving.value = false
  }
}
</script>
