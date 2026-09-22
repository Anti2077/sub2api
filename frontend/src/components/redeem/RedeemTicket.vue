<template>
  <section class="ticket-scene">
    <form ref="paper" class="ticket" :class="{ 'ticket-invalid': invalid }" @submit.prevent="requestRedeem">
      <div class="paper ticket-body" :class="{ 'ticket-torn': complete }">
        <div class="ticket-top">
          <span>CREDIT VOUCHER</span><span>ONE USE ONLY</span>
        </div>
        <h2>{{ t('redeem.ticketTitle') }}</h2>
        <div class="ticket-entry">
          <input
            id="code"
            ref="input"
            :value="complete ? redeemedCode : modelValue"
            required
            autocomplete="off"
            spellcheck="false"
            :disabled="locked || pending || dragging || complete"
            :placeholder="t('redeem.redeemCodePlaceholder')"
            :aria-label="t('redeem.redeemCodeLabel')"
            @input="
              emit(
                'update:modelValue',
                ($event.target as HTMLInputElement).value,
              )
            "
          />
        </div>
        <div class="ticket-bottom">
          <div class="barcode" aria-hidden="true" />
          <span :key="complete ? 'used' : 'waiting'" :class="{ 'ticket-used-reveal': complete }">{{
            t(complete ? 'redeem.ticketUsed' : 'redeem.ticketWaiting')
          }}</span>
        </div>
        <div v-if="complete" class="ticket-stamp" aria-hidden="true">
          {{ t('redeem.ticketStamp') }}<span>REDEEMED</span>
        </div>
      </div>
      <div class="ticket-seam" aria-hidden="true">
        <span>{{ t('redeem.ticketSeam') }}</span>
      </div>
      <!-- Separate attached/free layers keep the lower paper joined while the tear advances. -->
      <div
        ref="attached"
        class="paper ticket-stub attached"
        aria-hidden="true"
        inert
      >
        <span class="stub-label">{{ t('redeem.ticketStub') }}</span>
        <div class="stub-grip">
          <span>{{ t('redeem.ticketGrip') }}</span
          ><span class="stub-arrow">↘</span
          ><small>{{ t('redeem.ticketPull') }}</small>
        </div>
        <span class="stub-foot">TEAR HERE<br />ONE-TIME CREDIT</span>
      </div>
      <div
        ref="stub"
        class="paper ticket-stub"
        :class="{ 'stub-gone': complete }"
        role="button"
        :tabindex="complete ? -1 : 0"
        :aria-label="t('redeem.ticketAccessible')"
        :aria-disabled="locked || pending || complete"
        aria-describedby="ticket-help"
        @pointerdown="pointerDown"
        @pointermove="pointerMove"
        @pointerup="pointerEnd"
        @pointercancel="pointerEnd"
        @lostpointercapture="pointerEnd"
        @keydown="keyDown"
        @keyup="keyUp"
        @blur="cancelKeyboard"
      >
        <span class="stub-label">{{ t('redeem.ticketStub') }}</span>
        <div class="stub-grip">
          <span>{{
            t(pending ? 'redeem.redeeming' : 'redeem.ticketGrip')
          }}</span
          ><span class="stub-arrow">↘</span
          ><small>{{ t('redeem.ticketPull') }}</small>
        </div>
        <span class="stub-foot">TEAR HERE<br />ONE-TIME CREDIT</span>
      </div>
    </form>
    <p id="ticket-help" class="ticket-help" :class="{ 'ticket-used-reveal': complete }" role="status">
      {{
        t(
          invalid ? 'redeem.ticketInvalid' : pending
            ? 'redeem.redeeming'
            : complete
              ? 'redeem.ticketUsed'
              : 'redeem.ticketHelp',
        )
      }}
    </p>
    <button
      v-if="complete"
      type="button"
      class="btn btn-secondary mx-auto block"
      :disabled="locked"
      @click="reset"
    >
      {{ t('redeem.ticketAnother') }}
    </button>
  </section>
</template>
<script setup lang="ts">
import { onBeforeUnmount, onMounted, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ modelValue: string; locked: boolean; invalid?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [string]; redeem: [] }>()
const { t } = useI18n()
const paper = ref<HTMLElement>()
let shake: Animation | undefined
const input = ref<HTMLInputElement>(),
  stub = ref<HTMLElement>(),
  attached = ref<HTMLElement>()
const redeemedCode = ref('')
const pending = ref(false),
  complete = ref(false),
  dragging = ref(false)
let touchThreshold = 120
let pointer: number | null = null,
  origin = { x: 0, y: 0 },
  progress = 0,
  raf = 0,
  keyStart: number | null = null,
  settle: (() => void) | undefined
const clamp = (x: number) => Math.max(0, Math.min(1, x))
const ease = (x: number) => 1 - (1 - clamp(x)) ** 3
const reduce = () =>
  !window.matchMedia ||
  window.matchMedia('(prefers-reduced-motion: reduce)').matches ||
  document.hidden
function render(p: number) {
  if (!stub.value || !attached.value) return
  stub.value.style.opacity = '1'
  const front = Math.max(0.001, p) * 100
  attached.value.style.opacity = '1'
  attached.value.style.clipPath = `polygon(0 ${front}%,100% ${front}%,100% 100%,0 100%)`
  const points = [`0% ${front}%`]
  for (let i = 26; i >= 0; i--)
    points.push(`${i % 2 ? 2 : 0}% ${(i / 26) * front}%`)
  points.push('100% 0%', `100% ${front}%`)
  stub.value.style.clipPath = `polygon(${points.join(',')})`
  stub.value.style.transformOrigin = `0 ${front}%`
  stub.value.style.transform = `translate(${p * 12}px,${p * 8}px) rotate(${p * 17}deg)`
}
function releasePointer() {
  const id = pointer
  pointer = null
  if (id !== null && stub.value?.hasPointerCapture?.(id))
    stub.value.releasePointerCapture(id)
  keyStart = null
  dragging.value = false
}
function clearPaper() {
  stub.value?.removeAttribute('style')
  attached.value?.removeAttribute('style')
  progress = 0
}
function reset() {
  cancelAnimationFrame(raf)
  releasePointer()
  settle?.()
  settle = undefined
  pending.value = false
  flight = undefined
  fallen = false
  complete.value = false
  clearPaper()
  void nextTick(() => input.value?.focus())
}
function rollback() {
  settle?.()
  flight = undefined
  fallen = false
  cancelAnimationFrame(raf)
  releasePointer()
  pending.value = false
  if (reduce()) {
    clearPaper()
    return
  }
  const from = progress,
    start = performance.now()
  const frame = (now: number) => {
    const p = clamp((now - start) / 240)
    render(from * (1 - ease(p)))
    if (p < 1) raf = requestAnimationFrame(frame)
    else clearPaper()
  }
  raf = requestAnimationFrame(frame)
}
function indicateInvalid() {
  shake?.cancel()
  if (!reduce()) shake = paper.value?.animate?.([
    { transform: 'translateX(0)' }, { transform: 'translateX(-3px)' },
    { transform: 'translateX(3px)' }, { transform: 'translateX(-2px)' },
    { transform: 'translateX(2px)' }, { transform: 'translateX(0)' }
  ], { duration: 230, easing: 'ease-out' })
}
watch(() => props.invalid, value => {
  if (value) { settle?.(); flight = undefined; fallen = false; cancelAnimationFrame(raf); releasePointer(); pending.value = false; clearPaper(); void nextTick(indicateInvalid) }
})
function allowed() {
  if (props.invalid) { indicateInvalid(); return false }
  if (props.locked || pending.value || complete.value) return false
  if (!props.modelValue.trim()) {
    input.value?.focus()
    input.value?.reportValidity()
    return false
  }
  return true
}
let flight: Promise<void> | undefined
let fallen = false
function requestRedeem() {
  if (!allowed()) return
  cancelAnimationFrame(raf)
  releasePointer()
  progress = 1
  render(progress)
  pending.value = true
  flight = detach()
  emit('redeem')
}
function pointerDown(e: PointerEvent) {
  if (e.button !== 0 || pointer !== null || !allowed()) return
  cancelAnimationFrame(raf)
  const viewport = window.visualViewport
  const available = (viewport?.height ?? window.innerHeight) + (viewport?.offsetTop ?? 0) - e.clientY
  touchThreshold = e.pointerType === 'touch' ? Math.max(44, Math.min(80, available * .65)) : 120
  pointer = e.pointerId
  origin = { x: e.clientX, y: e.clientY }
  progress = 0
  dragging.value = true
  stub.value?.setPointerCapture?.(pointer)
}
function pointerMove(e: PointerEvent) {
  if (e.pointerId !== pointer) return
  progress = clamp(
    (Math.max(0, e.clientX - origin.x) * 0.65 +
      Math.max(0, e.clientY - origin.y)) /
      touchThreshold,
  )
  render(progress)
  if (progress >= 1) requestRedeem()
}
function pointerEnd(e: PointerEvent) {
  if (e.pointerId === pointer) rollback()
}
function keyDown(e: KeyboardEvent) {
  if (e.key !== ' ' && e.key !== 'Enter') return
  e.preventDefault()
  if (e.repeat || !allowed()) return
  cancelAnimationFrame(raf)
  keyStart = performance.now()
  dragging.value = true
  const frame = (now: number) => {
    if (keyStart === null) return
    progress = clamp((now - keyStart) / 650)
    render(progress)
    if (progress < 1) raf = requestAnimationFrame(frame)
  }
  raf = requestAnimationFrame(frame)
}
function keyUp(e: KeyboardEvent) {
  if ((e.key === ' ' || e.key === 'Enter') && keyStart !== null) {
    e.preventDefault()
    if (progress >= 1) requestRedeem()
    else rollback()
  }
}
function cancelKeyboard() {
  if (keyStart !== null) rollback()
}
function detach(): Promise<void> {
  cancelAnimationFrame(raf)
  releasePointer()
  progress = 1
  render(1)
  return new Promise((resolve) => {
    const done = () => {
      cancelAnimationFrame(raf)
      fallen = true
      if (stub.value) stub.value.style.opacity = '0'
      if (attached.value) attached.value.style.opacity = '0'
      settle = undefined
      resolve()
    }
    settle = done
    if (reduce()) {
      done()
      return
    }
    const start = performance.now()
    const frame = (now: number) => {
      const p = clamp((now - start) / 550)
      if (attached.value) attached.value.style.opacity = '0'
      if (stub.value) {
        stub.value.style.transformOrigin = '0 100%'
        stub.value.style.transform = `translate(${12 + p * 65}px,${8 + p * p * 85}px) rotate(${17 + p * 16}deg)`
        stub.value.style.opacity = String(1 - ease(clamp((p - 0.25) / 0.75)))
      }
      if (p < 1) raf = requestAnimationFrame(frame)
      else done()
    }
    raf = requestAnimationFrame(frame)
  })
}
async function accept(): Promise<void> {
  redeemedCode.value = props.modelValue
  await (flight ?? (fallen ? Promise.resolve() : detach()))
  pending.value = false
  complete.value = true
  clearPaper()
  flight = undefined
}
watch(
  () => props.locked,
  (value) => {
    if (!value && pending.value) rollback()
  },
)
function visibilityChange() {
  if (document.hidden) {
    if (settle) settle()
    else if (dragging.value) rollback()
  }
}
onMounted(() => document.addEventListener('visibilitychange', visibilityChange))
onBeforeUnmount(() => {
  shake?.cancel()
  document.removeEventListener('visibilitychange', visibilityChange)
  cancelAnimationFrame(raf)
  releasePointer()
  settle?.()
})
defineExpose({ accept, rollback, reset })
</script>
<style scoped>
.ticket-scene {
  --paper: theme('colors.canvas.50');
  --ink: theme('colors.primary.800');
  --muted: theme('colors.primary.500');
  --rule: theme('colors.primary.200');
  padding: 12px 8px 26px;
  overflow: clip;
  color: var(--ink);
}
:global(.dark .ticket-scene) {
  --paper: theme('colors.dark.900');
  --ink: theme('colors.primary.100');
  --muted: theme('colors.dark.200');
  --rule: theme('colors.dark.600');
}
.ticket {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 126px;
  position: relative;
  filter: drop-shadow(0 12px 15px #17253712);
  isolation: isolate;
}
.paper {
  background: var(--paper);
  background-image: radial-gradient(#89919612 0.6px, transparent 0.6px);
  background-size: 5px 5px;
}
.ticket-body {
  grid-row: 1;
  grid-column: 1;
  padding: 24px 25px 19px;
  position: relative;
  border-radius: 10px 0 0 10px;
  min-width: 0;
  border-right: 1px dashed var(--rule);
}
.ticket-top {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--muted);
  font:
    8px ui-monospace,
    monospace;
  letter-spacing: 0.06em;
}
.ticket-body h2 {
  font-size: 19px;
  font-weight: 600;
  margin: 22px 0 8px;
}
.ticket-description {
  font-size: 11px;
  color: var(--muted);
}
.ticket-entry {
  margin-top: 28px;
}
.ticket-entry label {
  display: block;
  font-size: 11px;
  color: var(--muted);
}
.ticket-entry input {
  width: 100%;
  display: block;
  background: transparent;
  border: 0;
  border-bottom: 1px dashed var(--rule);
  border-radius: 0;
  padding: 14px 0 12px;
  min-height: 52px;
  color: var(--ink);
  font:
    20px ui-monospace,
    monospace;
  letter-spacing: 0;
  outline: none;
}
.ticket-entry input:focus {
  border-bottom: 2px solid var(--muted);
}
.ticket-entry input::placeholder {
  color: var(--muted);
  font-size: 14px;
  letter-spacing: 0;
}
.ticket-entry small {
  display: block;
  font-size: 10px;
  color: var(--muted);
  margin-top: 7px;
}
.ticket-bottom {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 8px;
  margin-top: 19px;
  font-size: 10px;
  color: var(--muted);
}
.barcode {
  height: 20px;
  width: 95px;
  background: repeating-linear-gradient(
    90deg,
    var(--muted) 0px,
    var(--muted) 1px,
    transparent 1px,
    transparent 3px,
    var(--muted) 3px,
    var(--muted) 5px,
    transparent 5px,
    transparent 7px,
    var(--muted) 7px,
    var(--muted) 8px,
    transparent 8px,
    transparent 12px
  );
  opacity: 0.55;
}
.ticket-seam {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 125px;
  z-index: 4;
  pointer-events: none;
  border-right: 1px dashed var(--rule);
}
.ticket-seam span {
  position: absolute;
  top: 38%;
  left: -8px;
  writing-mode: vertical-rl;
  font-size: 10px;
  letter-spacing: 2px;
  color: var(--muted);
  background: var(--paper);
  padding: 7px 3px;
  white-space: nowrap;
}
.ticket-stub {
  grid-row: 1;
  grid-column: 2;
  display: flex;
  align-items: center;
  flex-direction: column;
  justify-content: space-between;
  padding: 24px 12px 20px;
  border-radius: 0 10px 10px 0;
  position: relative;
  z-index: 2;
  transform-origin: 0 100%;
  will-change: transform, opacity;
  touch-action: none;
  user-select: none;
  cursor: grab;
}
.ticket-stub:active {
  cursor: grabbing;
}
.ticket-stub:focus-visible {
  outline: 2px solid var(--muted);
  outline-offset: 3px;
}
.attached {
  z-index: 1;
  opacity: 0;
  pointer-events: none;
}
.stub-gone {
  visibility: hidden;
}
.stub-label {
  font-size: 10px;
  letter-spacing: 0.1em;
  color: var(--muted);
}
.stub-grip {
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 10px;
  text-align: center;
  font-size: 16px;
  font-weight: 600;
  pointer-events: none;
}
.stub-arrow {
  font-size: 26px;
  font-weight: 300;
}
.stub-grip small {
  font-size: 10px;
  font-weight: 400;
  color: var(--muted);
}
.stub-foot {
  font:
    8px/1.8 ui-monospace,
    monospace;
  text-align: center;
  color: var(--muted);
}
.ticket-stamp {
  position: absolute;
  right: 25px;
  top: 70px;
  border: 2px solid currentColor;
  color: theme('colors.emerald.600');
  border-radius: 4px;
  padding: 8px 12px;
  text-align: center;
  font-size: 20px;
  font-weight: 650;
  transform: rotate(-12deg);
  transform-origin: 50% 55%;
  animation: ticket-stamp-press 520ms both;
  background: var(--paper);
  pointer-events: none;
}
.ticket-stamp span {
  display: block;
  font:
    8px ui-monospace,
    monospace;
  margin-top: 4px;
}
.ticket-help {
  text-align: center;
  color: var(--muted);
  font-size: 11px;
  margin: 22px 0 12px;
}
@media (max-width: 550px) {
  .ticket {
    grid-template-columns: minmax(0, 1fr) 86px;
  }
  .ticket-body {
    padding: 19px 16px;
  }
  .ticket-body h2 {
    font-size: 16px;
  }
  .ticket-top span:last-child {
    display: none;
  }
  .ticket-entry input {
    font-size: 16px;
    min-height: 44px;
    letter-spacing: 0;
  }
  .ticket-stub {
    padding: 20px 6px;
  }
  .ticket-seam {
    right: 85px;
  }
  .stub-label,
  .stub-grip small {
    font-size: 9px;
  }
  .stub-grip {
    font-size: 13px;
  }
  .ticket-stamp {
    right: 15px;
    font-size: 16px;
  }
  .barcode {
    width: 55px;
  }
}
.ticket-torn {
  border-right: 0;
  clip-path: polygon(
    0 0,
    100% 0,
    99% 2%,
    100% 4%,
    99% 6%,
    100% 8%,
    99% 10%,
    100% 12%,
    99% 14%,
    100% 16%,
    99% 18%,
    100% 20%,
    99% 22%,
    100% 24%,
    99% 26%,
    100% 28%,
    99% 30%,
    100% 32%,
    99% 34%,
    100% 36%,
    99% 38%,
    100% 40%,
    99% 42%,
    100% 44%,
    99% 46%,
    100% 48%,
    99% 50%,
    100% 52%,
    99% 54%,
    100% 56%,
    99% 58%,
    100% 60%,
    99% 62%,
    100% 64%,
    99% 66%,
    100% 68%,
    99% 70%,
    100% 72%,
    99% 74%,
    100% 76%,
    99% 78%,
    100% 80%,
    99% 82%,
    100% 84%,
    99% 86%,
    100% 88%,
    99% 90%,
    100% 92%,
    99% 94%,
    100% 96%,
    99% 98%,
    100% 100%,
    0 100%
  );
}
.ticket-invalid .ticket-stub{cursor:not-allowed}.ticket-invalid .ticket-entry input{border-bottom-color:theme('colors.red.400')}.ticket-invalid + .ticket-help{color:theme('colors.red.500')}

/* The impression lands first; the receipt status follows the impact. */
@keyframes ticket-stamp-press {
  0% { opacity: 0; transform: translate(6px, -20px) rotate(-19deg) scale(1.55); }
  25% { opacity: 1; transform: translate(2px, -9px) rotate(-15deg) scale(1.24); }
  48% { opacity: 1; transform: translate(0, 2px) rotate(-12deg) scale(.96, .92); }
  68% { transform: translate(0, -1px) rotate(-11deg) scale(1.035); }
  84% { transform: translate(0, .5px) rotate(-12.5deg) scale(.995); }
  100% { opacity: 1; transform: translate(0, 0) rotate(-12deg) scale(1); }
}
.ticket-used-reveal { animation: ticket-status-in 260ms 260ms both; }
@keyframes ticket-status-in {
  from { opacity: 0; transform: translateY(3px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) {
  .ticket-stamp, .ticket-used-reveal { animation: none; }
}

@media (max-width: 550px) {
  .ticket-entry input::placeholder { font-size: 16px; }
  .ticket-bottom { flex-wrap: wrap; }
  .ticket-body.ticket-torn { padding-bottom: 78px; }
  .ticket-stamp { top: auto; bottom: 12px; right: 16px; }
  .ticket-help { font-size: 12px; line-height: 1.6; }
}
@media (max-width: 360px) {
  .ticket { grid-template-columns: minmax(0, 1fr) 76px; }
  .ticket-seam { right: 75px; }
  .ticket-body { padding-inline: 12px; }
  .ticket-body h2 { overflow-wrap: anywhere; }
}

@media (max-width: 550px) {
  .ticket-entry { margin-top: 22px; }
}
</style>
