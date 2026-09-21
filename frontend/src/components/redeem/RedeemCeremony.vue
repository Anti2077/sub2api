<template>
  <Teleport to="body">
    <canvas
      v-if="active"
      ref="canvas"
      class="celebration-canvas"
      aria-hidden="true"
    />
  </Teleport>
</template>
<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { startCeremony } from './ceremony'
const emit = defineEmits<{ progress: [number]; active: [boolean] }>()
const canvas = ref<HTMLCanvasElement>()
const active = ref(false)
let cancel: (() => void) | undefined
let generation = 0
function stop() {
  generation++
  cancel?.()
  cancel = undefined
  active.value = false
  emit('active', false)
}
async function play() {
  stop()
  const token = generation
  if (
    !window.matchMedia ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches ||
    document.hidden
  ) {
    emit('progress', 1)
    return
  }
  active.value = true
  emit('active', true)
  await nextTick()
  if (token !== generation) return
  await new Promise<void>((resolve) => {
    const done = () => {
      active.value = false
      emit('active', false)
      cancel = undefined
      resolve()
    }
    if (!canvas.value) {
      emit('progress', 1)
      done()
      return
    }
    try {
      cancel = startCeremony({
        canvas: canvas.value,
        onProgress: (p) => emit('progress', p),
        onDone: done,
      })
    } catch {
      emit('progress', 1)
      done()
    }
  })
}
onBeforeUnmount(stop)
defineExpose({ play, stop })
</script>
<style scoped>
.celebration-canvas {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 60;
}
</style>
