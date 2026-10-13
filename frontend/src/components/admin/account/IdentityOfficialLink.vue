<template>
  <a v-if="url" :href="url" target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer" class="btn btn-ghost min-h-11 text-xs" data-testid="identity-official-link">
    {{ t(isActive(run) ? 'admin.modelIdentity.officialProgress' : 'admin.modelIdentity.officialReport') }} ↗
  </a>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { IdentityRun } from '@/api/admin/modelIdentity'
import { identityOfficialURL, isActive } from '@/utils/modelIdentity'

const props = defineProps<{ run?: IdentityRun | null }>()
const { t } = useI18n()
const url = computed(() => identityOfficialURL(props.run))
</script>
