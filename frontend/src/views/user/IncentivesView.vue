<template>
  <AppLayout>
    <div class="reward-page">
      <RewardsNav current="benefits" />
      <header class="rw-header">
        <div>
          <p class="rw-eyebrow">
            WEEKLY PERKS / 01
          </p><h1>
            {{ t('incentives.title') }}<span class="rw-title-mark">
              .
            </span>
          </h1><p>
            {{ t('incentives.description') }}
          </p>
        </div><div class="benefit-stamp">
          <RewardIcon name="gift" /><span>
            {{ t('rewards.intoPocket') }}
          </span>
        </div>
      </header>
      <p v-if="error"
         role="alert"
         class="rw-alert">
        <RewardIcon name="shield" />{{ error }}<button type="button"
                                                       class="rw-link"
                                                       @click="load()">
          {{ t('incentives.refresh') }}
        </button>
      </p><p v-if="loading"
             role="status"
             class="rw-caption">
        {{ t('incentives.loading') }}
      </p>
      <div v-if="loading && !items.length"
           class="benefit-skeleton"
           aria-hidden="true">
        <div/><div/>
      </div>
      <div class="benefit-flow">
        <div :class="['benefit-step', {'is-done':lottery?.checked_in_today,'is-current':!lottery?.checked_in_today}]">
          <span>
            01
          </span><RewardIcon name="check" /><div>
            <strong>
              {{ t('rewards.checkInStep') }}
            </strong><small>
              {{ t('rewards.checkInStepNote') }}
            </small>
          </div>
        </div><i>
          →
        </i><div :class="['benefit-step', {'is-current':lottery?.checked_in_today}]">
          <span>
            02
          </span><RewardIcon name="gift" /><div>
            <strong>
              {{ t('rewards.drawStep') }}
            </strong><small>
              {{ t('rewards.drawStepNote') }}
            </small>
          </div>
        </div><i>
          →
        </i><div class="benefit-step">
          <span>
            03
          </span><RewardIcon name="ticket" /><div>
            <strong>
              {{ t('rewards.cardStep') }}
            </strong><small>
              {{ t('rewards.cardStepNote') }}
            </small>
          </div>
        </div>
      </div>
      <div v-if="items.length"
           class="benefit-grid">
        <section v-for="item in items"
                 :id="item.kind"
                 :key="item.kind"
                 class="rw-panel benefit-card"
                 :class="`benefit-${item.kind}`">
          <div class="benefit-card-head">
            <div class="benefit-icon">
              <RewardIcon :name="item.kind==='global_rate'?'spark':'gift'" />
            </div><div>
              <p class="rw-eyebrow">
                {{ item.kind==='global_rate' ? 'SITEWIDE RATE' : 'DAILY DRAW' }}
              </p><h2>
                {{ t(`incentives.${item.kind}`) }}
              </h2>
            </div><span class="benefit-status"
                        :class="{active:isItemActive(item)}">
              <i />{{ isItemActive(item) ? t('incentives.enabled') : t('incentives.disabled') }}
            </span>
          </div>
          <p class="rw-caption benefit-period">
            {{ date(item.starts_at) }} — {{ date(item.ends_at) }} · {{ item.timezone }}
          </p><template v-if="item.kind==='global_rate'">
            <div v-if="item.enabled"
                 class="benefit-rate">
              <div>
                <span class="rw-caption">
                  {{ t('incentives.progress') }}
                </span><strong>
                  ${{ item.spend.toFixed(2) }}
                </strong>
              </div><div class="rate-next">
                <span>
                  {{ t('incentives.next') }}
                </span><strong>
                  ${{ item.next_threshold.toFixed(2) }}
                </strong>
              </div>
            </div><div v-if="item.enabled"
                       class="rw-progress">
              <span :style="{width:`${progressPercent(item)}%`}" />
            </div><div v-if="item.groups.length"
                       class="rate-list">
              <div v-for="group in item.groups"
                   :key="group.group_id">
                <span>
                  {{ group.name }}
                </span><span class="rate-values">
                  <del>
                    {{ formatRate(group.base_rate) }}×
                  </del><strong>
                    {{ formatRate(group.current_rate) }}×
                  </strong>
                </span>
              </div>
            </div><p v-if="item.eligible"
                     class="rw-note">
              <RewardIcon name="spark" />{{ t('incentives.rateNote') }}
            </p><p v-else
                   class="benefit-muted">
              {{ t('incentives.excludedNotice') }}
            </p><p v-if="!item.enabled"
                   class="benefit-muted">
              {{ t('incentives.disabled') }}
            </p>
          </template>
          <template v-else>
            <div class="lottery-welcome">
              <svg class="lottery-gift" viewBox="0 0 120 120" aria-hidden="true"><path d="M19 48h82v54H19z" fill="#e8b96d"/><path d="M14 35h92v20H14z" fill="#d88e61"/><path d="M54 35h12v67H54z" fill="#f6df9d"/><path d="M60 35c-20 0-30-9-24-19 6-10 24 0 24 19Zm0 0c20 0 30-9 24-19-6-10-24 0-24 19Z" fill="#f0c780" stroke="#b87754" stroke-width="3"/><path d="M19 102h82" stroke="#b87754" stroke-width="5" stroke-linecap="round"/></svg>
              <div><h3>{{ t('rewards.drawWelcome') }}</h3><p>{{ t('rewards.drawWelcomeNote') }}</p></div>
            </div>
            <div class="lottery-balance">
              <div>
                <span class="rw-caption">
                  {{ t('incentives.available') }}
                </span><strong>
                  {{ item.available.toString().padStart(2,'0') }}
                </strong><small>
                  {{ t('rewards.drawUnit') }}
                </small>
              </div><div class="lottery-totals">
                <span>
                  {{ t('incentives.earned') }} <b>
                    {{ item.earned }}
                  </b>
                </span><span>
                  {{ t('incentives.used') }} <b>
                    {{ item.used }}
                  </b>
                </span>
              </div>
            </div><div class="checkin-box">
              <div>
                <strong>
                  {{ t('incentives.checkInTitle') }}
                </strong><p>
                  {{ t('incentives.checkInDescription') }}
                </p>
              </div><button type="button"
                            data-testid="incentive-check-in"
                            class="btn btn-secondary"
                            :disabled="checkInPending||item.checked_in_today||!item.check_in_enabled"
                            @click="checkIn">
                {{ checkInPending?t('incentives.checkingIn'):item.checked_in_today?t('incentives.checkedInToday'):item.check_in_enabled?t('incentives.checkIn'):t('incentives.checkInDisabled') }}
              </button>
            </div><div v-if="drawing"
                       data-testid="incentive-draw-animation"
                       class="draw-stage"
                       role="status"
                       aria-live="polite"
                       aria-busy="true">
              <div class="draw-orbit">
                <RewardIcon name="spark" />
              </div><div>
                <p class="rw-eyebrow">
                  {{ t('incentives.drawing') }}
                </p><strong>
                  {{ activePrize?.name || t('incentives.drawing') }}
                </strong>
              </div><div class="draw-dots">
                <i v-for="prize in enabledPrizes"
                   :key="prize.id"
                   :class="{'is-active':activePrize?.id===prize.id}" />
              </div>
            </div><button type="button"
                          data-testid="incentive-draw"
                          class="draw-button"
                          :disabled="drawing||item.available<1||(!item.enabled&&!item.check_in_enabled)"
                          @click="draw">
              <RewardIcon name="gift" />{{ t(drawing?'incentives.drawing':'incentives.draw') }}<span>
                {{ item.available }}
              </span>
            </button><p v-if="!drawing && item.available < 1" class="draw-empty-hint">{{ t('rewards.drawEmptyHint') }}</p><p v-if="result"
                        role="status"
                        class="draw-result">
              <RewardIcon name="check" /><span>{{ result.prize.name }} · {{ result.prize.card?.name || '$'+result.reward_amount.toFixed(2) }}<small>{{ result.prize.card ? t('rewards.cardNote') : t('rewards.balanceDeposited') }}</small></span><RouterLink v-if="result.prize.card" class="rw-link" to="/cards?tab=bag">{{ t('rewards.viewBag') }}<RewardIcon name="arrow" /></RouterLink>
            </p><p class="rw-caption mt-3">
              {{ t('incentives.expiry') }}
            </p><div v-if="item.enabled"
                     class="lottery-consumption">
              <p class="rw-caption">
                {{ t('incentives.progress') }} ${{ item.personal_spend.toFixed(2) }} · {{ t('incentives.next') }} ${{ item.next_threshold.toFixed(2) }}
              </p><div class="rw-progress"
                       role="progressbar"
                       :aria-label="t('incentives.progress')"
                       :aria-valuenow="progressPercent(item)"
                       aria-valuemin="0"
                       aria-valuemax="100">
                <span :style="{width:`${progressPercent(item)}%`}" />
              </div>
            </div><p v-else
                     class="rw-caption mt-3">
              {{ t('incentives.consumptionDisabled') }}
            </p><div v-if="enabledPrizes.length"
                     class="prize-grid">
              <p>
                {{ t('incentives.prizes') }}
              </p><div v-for="prize in enabledPrizes"
                       :key="prize.id"
                       class="prize-chip"
                       :class="{'is-active':drawing&&activePrize?.id===prize.id,'is-winning':!drawing&&result?.prize.id===prize.id}">
                <RewardIcon :name="prize.card?'ticket':'crystal'" /><span>
                  {{ prize.name }}<small>
                    {{ prize.card?.name || '$'+prize.reward_amount.toFixed(2) }}
                  </small>
                </span><b>
                  {{ (prize.probability*100).toFixed(2) }}%
                </b>
              </div>
            </div>
          </template>
        </section>
      </div>
      <section v-if="history"
               class="rw-panel benefit-history">
        <div class="rw-section-title">
          <div>
            <h2>
              {{ t('incentives.history') }}
            </h2><p class="rw-caption">
              {{ t('rewards.historyNote') }}
            </p>
          </div><RewardIcon name="clock" />
        </div><p v-if="!history.rewards.length"
                 class="rw-empty">
          {{ t('incentives.empty') }}
        </p><ul v-else>
          <li v-for="reward in history.rewards"
              :key="reward.id">
            <span class="history-dot" /><time>
              {{ date(reward.created_at) }}
            </time><strong>
              {{ reward.prize.name }}
            </strong><span>
              {{ reward.prize.card?.name || '$'+reward.reward_amount.toFixed(2) }}
            </span>
          </li>
        </ul>
      </section>
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import RewardsNav from '@/components/rewards/RewardsNav.vue'
import RewardIcon from '@/components/rewards/RewardIcon.vue'
import { incentivesAPI, type IncentiveHistory, type IncentiveStatus } from '@/api/incentives'
import type { DailyLotteryPrize } from '@/api/dailyLottery'

const { t } = useI18n()
const items = ref<IncentiveStatus[]>([])
const history = ref<IncentiveHistory>()
const loading = ref(false)
const drawing = ref(false)
const checkInPending = ref(false)
const error = ref('')
const result = ref<IncentiveDrawResult>()
const activePrizeIndex = ref(-1)
let requestKey: string | null = null
let prizeAnimationTimer: number | undefined

const drawAnimationDuration = 1350

const lottery = computed(() => items.value.find((item) => item.kind === 'lottery'))
const enabledPrizes = computed(() => lottery.value?.prizes.filter((prize) => prize.enabled) ?? [])
const activePrize = computed(() => enabledPrizes.value[activePrizeIndex.value] ?? null)

const date = (value: string) => new Date(value).toLocaleString()
const formatRate = (value: number) => (Number.isFinite(value) ? value.toFixed(4) : '0.0000')
const progressPercent = (item: IncentiveStatus) => {
  if (item.threshold <= 0) return 0
  const progress = item.threshold - item.next_threshold
  return Math.min(100, Math.max(0, (progress / item.threshold) * 100))
}
const isItemActive = (item: IncentiveStatus) => item.enabled || (item.kind === 'lottery' && item.check_in_enabled)

type IncentiveDrawResult = {
  prize: DailyLotteryPrize
  reward_amount: number
  source?: 'checkin' | 'consumption'
}

type LoadOptions = {
  silent?: boolean
}

function errorMessage(value: unknown): string {
  if (value instanceof Error && value.message) return value.message
  if (typeof value === 'object' && value !== null && 'message' in value) {
    const message = String((value as { message?: unknown }).message || '')
    if (message) return message
  }
  if (typeof value === 'object' && value !== null && 'response' in value) {
    const response = (value as { response?: { data?: { message?: unknown } } }).response
    const message = String(response?.data?.message || '')
    if (message) return message
  }
  return t('incentives.error')
}

async function load(options: LoadOptions = {}) {
  const { silent = false } = options
  if (!silent) {
    loading.value = true
    error.value = ''
  }

  const [statusResult, historyResult] = await Promise.allSettled([
    incentivesAPI.status(),
    incentivesAPI.history()
  ])

  if (statusResult.status === 'fulfilled') {
    items.value = statusResult.value
  }
  if (historyResult.status === 'fulfilled') {
    history.value = historyResult.value
  }

  if (!silent) {
    const failed = statusResult.status === 'rejected' ? statusResult.reason : historyResult.status === 'rejected' ? historyResult.reason : null
    if (failed) error.value = errorMessage(failed)
    loading.value = false
  }
}

async function checkIn() {
  if (checkInPending.value || !lottery.value?.check_in_enabled || lottery.value.checked_in_today) return
  checkInPending.value = true
  error.value = ''
  try {
    const checkInResult = await incentivesAPI.checkIn()
    const current = lottery.value
    if (current) {
      current.checked_in_today = true
      if (checkInResult.chance_awarded) {
        current.check_in_chance_awarded = true
        current.earned += 1
        current.available += 1
      }
    }
    await load({ silent: true })
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    checkInPending.value = false
  }
}

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function stopPrizeAnimation() {
  if (prizeAnimationTimer !== undefined) {
    window.clearTimeout(prizeAnimationTimer)
    prizeAnimationTimer = undefined
  }
}

function startPrizeAnimation() {
  stopPrizeAnimation()
  activePrizeIndex.value = enabledPrizes.value.length > 0 ? Math.floor(Math.random() * enabledPrizes.value.length) : -1
  if (prefersReducedMotion() || enabledPrizes.value.length <= 1) return

  const tick = () => {
    if (!drawing.value || enabledPrizes.value.length <= 1) return
    const current = activePrizeIndex.value
    let next = Math.floor(Math.random() * enabledPrizes.value.length)
    while (next === current) next = Math.floor(Math.random() * enabledPrizes.value.length)
    activePrizeIndex.value = next
    prizeAnimationTimer = window.setTimeout(tick, 75 + Math.floor(Math.random() * 70))
  }

  prizeAnimationTimer = window.setTimeout(tick, 90)
}

function applyDrawResult(drawResult: IncentiveDrawResult) {
  const current = lottery.value
  if (!current) return
  current.available = Math.max(0, current.available - 1)
  current.used += 1
  if (drawResult.source === 'checkin') {
    current.checked_in_today = true
    current.check_in_chance_awarded = true
  }
}

function waitForAnimation(startedAt: number) {
  if (prefersReducedMotion()) return Promise.resolve()
  const remaining = Math.max(0, drawAnimationDuration - (Date.now() - startedAt))
  return remaining > 0 ? new Promise<void>((resolve) => window.setTimeout(resolve, remaining)) : Promise.resolve()
}

async function draw() {
  if (drawing.value || !lottery.value || lottery.value.available < 1) return
  drawing.value = true
  error.value = ''
  result.value = undefined
  const startedAt = Date.now()
  startPrizeAnimation()
  requestKey ??= crypto.randomUUID()

  try {
    const drawResult = await incentivesAPI.draw(requestKey)
    await waitForAnimation(startedAt)
    stopPrizeAnimation()
    activePrizeIndex.value = enabledPrizes.value.findIndex((prize) => prize.id === drawResult.prize.id)
    result.value = drawResult
    applyDrawResult(drawResult)
    requestKey = null
    // The draw is already committed. A refresh failure must not turn a successful draw into a blocking page error.
    await load({ silent: true })
  } catch (e) {
    stopPrizeAnimation()
    error.value = errorMessage(e)
  } finally {
    drawing.value = false
  }
}

onMounted(load)
onBeforeUnmount(stopPrizeAnimation)
</script>

<style scoped>
.incentive-draw-stage {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 74px;
  padding: 14px 16px;
  border: 1px solid rgb(129 140 248 / 0.35);
  border-radius: 14px;
  background: linear-gradient(135deg, rgb(238 242 255 / 0.9), rgb(240 253 250 / 0.9));
}

.incentive-draw-stage__pulse {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: rgb(99 102 241 / 0.14);
  animation: incentive-draw-pulse 900ms ease-in-out infinite;
}

.incentive-draw-stage__pulse span {
  width: 11px;
  height: 11px;
  border-radius: 9999px;
  background: rgb(79 70 229);
  box-shadow: 0 0 0 7px rgb(99 102 241 / 0.12);
}

.incentive-draw-stage__dots {
  display: flex;
  flex-shrink: 0;
  gap: 5px;
  margin-left: auto;
}

.incentive-draw-stage__dots span {
  width: 7px;
  height: 7px;
  border-radius: 9999px;
  background: rgb(148 163 184 / 0.5);
  transition: transform 120ms ease, background-color 120ms ease;
}

.incentive-draw-stage__dots span.is-active {
  transform: scale(1.45);
  background: rgb(79 70 229);
}

.incentive-prize-row {
  padding: 8px 10px;
  margin: 0 -10px;
  border-radius: 9px;
  transition: color 160ms ease, background-color 160ms ease, transform 160ms ease;
}

.incentive-prize-row--active {
  color: rgb(67 56 202);
  background: rgb(224 231 255 / 0.78);
  transform: translateX(3px);
}

.incentive-prize-row--winning {
  color: rgb(4 120 87);
  background: rgb(209 250 229 / 0.75);
}

@keyframes incentive-draw-pulse {
  0%, 100% { transform: scale(0.92); opacity: 0.7; }
  50% { transform: scale(1.08); opacity: 1; }
}

:global(.dark .incentive-draw-stage) {
  border-color: rgb(129 140 248 / 0.38);
  background: linear-gradient(135deg, rgb(30 41 99 / 0.45), rgb(6 78 59 / 0.3));
}

:global(.dark .incentive-draw-stage__pulse) {
  background: rgb(129 140 248 / 0.2);
}

:global(.dark .incentive-draw-stage__pulse span) {
  background: rgb(165 180 252);
  box-shadow: 0 0 0 7px rgb(129 140 248 / 0.15);
}

:global(.dark .incentive-draw-stage__dots span) {
  background: rgb(148 163 184 / 0.4);
}

:global(.dark .incentive-draw-stage__dots span.is-active) {
  background: rgb(165 180 252);
}

:global(.dark .incentive-prize-row--active) {
  color: rgb(199 210 254);
  background: rgb(49 46 129 / 0.35);
}

:global(.dark .incentive-prize-row--winning) {
  color: rgb(167 243 208);
  background: rgb(6 78 59 / 0.35);
}

@media (prefers-reduced-motion: reduce) {
  .incentive-draw-stage__pulse,
  .incentive-draw-stage__dots span,
  .incentive-prize-row {
    animation: none;
    transition: none;
  }
}
</style>
