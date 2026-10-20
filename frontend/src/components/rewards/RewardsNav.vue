<template>
  <nav class="rewards-nav" :aria-label="t('rewards.explore')">
    <RouterLink v-for="item in links" :key="item.path" :to="item.path" :class="{ selected: current === item.id }" :aria-current="current === item.id ? 'page' : undefined"><RewardIcon :name="item.icon" /><span>{{ t(`rewards.${item.id}`) }}</span><span class="nav-arrow" aria-hidden="true">↗</span></RouterLink>
  </nav>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import RewardIcon from './RewardIcon.vue'
defineProps<{ current: 'benefits' | 'cards' | 'farm' }>()
const { t } = useI18n()
const links = [{ id: 'benefits', path: '/incentives', icon: 'gift' }, { id: 'cards', path: '/cards', icon: 'ticket' }, { id: 'farm', path: '/farm', icon: 'leaf' }]
</script>
