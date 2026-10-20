<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="paths[name] || paths.spark" /></svg>
</template>
<script setup lang="ts">
withDefaults(defineProps<{ name?: string }>(), { name: 'spark' })
const paths: Record<string, string> = {
 leaf: 'M12 21V11m0 5C3 16 2 10 3 4c7-1 10 3 9 12Zm0-5c0-6 4-9 10-8 0 6-3 10-10 8Z',
 crystal: 'm12 2 8 10-8 10-8-10Zm0 0v20M4 12h16',
 basket: 'm3 10 3 11h12l3-11H3Zm5 0 4-8 4 8M9 14v3m6-3v3',
 friends: 'M12 8a3 3 0 1 1-6 0 3 3 0 0 1 6 0ZM2 21v-3a7 7 0 0 1 14 0v3M16 5a3 3 0 0 1 0 6m3 3a6 6 0 0 1 3 5v2',
 gift: 'M3 8h18v5H3zM5 13v8h14v-8M12 8v13m0-13C2 8 6 0 10 4l2 4Zm0 0c10 0 6-8 2-4l-2 4Z',
 arrow: 'M4 12h15m-6-6 6 6-6 6',
 clock: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM12 6v6l4 2',
 sun: 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM12 1v2m0 18v2M1 12h2m18 0h2M4 4l2 2m12 12 2 2M4 20l2-2M18 6l2-2',
 spark: 'm12 2 3 7 7 3-7 3-3 7-3-7-7-3 7-3Z',
 bag: 'M5 7h14l2 14H3L5 7Zm3 0V5a4 4 0 0 1 8 0v2',
 check: 'm5 12 4 4L20 5',
 shield: 'M12 2 3 6v6c0 5 9 10 9 10s9-5 9-10V6l-9-4Zm-4 10 3 3 5-6',
 refresh: 'M20 7v5h-5M4 17v-5h5m11 0A8 8 0 0 0 6 6l-2 2m0 4a8 8 0 0 0 14 6l2-2',
 ticket: 'M3 5h18v5a2 2 0 0 0 0 4v5H3v-5a2 2 0 0 0 0-4V5Zm12 0v3m0 3v2m0 3v3',
 home: 'm2 11 10-9 10 9M5 9v12h14V9m-10 12v-8h6v8',
 plus: 'M12 4v16M4 12h16',
 hand: 'M8 13V5a2 2 0 0 1 4 0v7-8a2 2 0 0 1 4 0v8-5a2 2 0 0 1 4 0v9c0 5-4 6-7 6-2 0-4-1-5-3l-5-6a2 2 0 0 1 3-3l2 3Z'
}
</script>
