<template>
  <AppLayout>
    <div class="space-y-4 pb-8">
      <RouterLink to="/admin/ops" class="inline-flex min-h-[40px] items-center text-sm font-medium text-primary-700 hover:underline dark:text-primary-300">← {{ t('admin.ops.routingMonitor.backToOverview') }}</RouterLink>
      <OpsRoutingMonitor :show-heading="false" />
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import OpsRoutingMonitor from './components/OpsRoutingMonitor.vue'
const { t } = useI18n()
</script>
