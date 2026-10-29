<template>
  <AppLayout>
    <div class="reward-page">
      <RewardsNav current="cards" />
      <header class="rw-header">
        <div>
          <p class="rw-eyebrow">
            YOUR REWARDS / 02
          </p><h1>
            {{ t('rewards.cards') }}<span class="rw-title-mark">
              .
            </span>
          </h1><p>
            {{ t('rewards.cardNote') }}
          </p>
        </div><div class="rw-stats">
          <div>
            <small>
              {{ t('rewards.unused') }}
            </small><strong>
              {{ unusedCount.toString().padStart(2,'0') }}
            </strong>
          </div><div>
            <small>
              {{ t('rewards.active') }}
            </small><strong>
              {{ activeCount.toString().padStart(2,'0') }}
            </strong>
          </div><div>
            <small>
              {{ t('rewards.availableTokens') }}
            </small><strong>
              {{ availableTokens.toLocaleString() }}
            </strong>
          </div>
        </div>
      </header>
      <div class="farm-toolbar">
        <nav class="rw-segments"
             :aria-label="t('rewards.cards')">
          <button v-for="v in tabs"
                  :key="v"
                  :aria-pressed="tab===v"
                  @click="select(v)">
            {{ t(`rewards.${v}`) }}<span v-if="v==='bag'">
              {{ cards.length }}
            </span>
          </button>
        </nav><button class="rw-link"
                      :disabled="busy||loading"
                      @click="load">
          <RewardIcon name="refresh" />{{ t('rewards.refresh') }}
        </button>
      </div>
      <p v-if="error"
         role="alert"
         class="rw-alert">
        {{ error }}
      </p><p v-if="notice"
             role="status"
             class="rw-success">
        <RewardIcon name="check" />{{ notice }}
      </p>
      <section v-if="tab==='redeem'"
               class="rw-panel cards-redeem">
        <div class="redeem-heading">
          <span class="ticket-symbol">
            <RewardIcon name="ticket" />
          </span><div>
            <p class="rw-eyebrow">
              A LITTLE SOMETHING FOR YOU
            </p><h2>
              {{ t('rewards.redeem') }}
            </h2><p class="rw-caption">
              {{ t('rewards.redeemHint') }}
            </p>
          </div>
        </div><RedeemView embedded @redeemed="load" />
        <button class="rw-link" @click="select('bag')">{{ t('rewards.viewBag') }}<RewardIcon name="arrow" /></button>
      </section>
      <template v-else-if="tab==='bag'">
        <nav v-if="cards.length" class="cards-filter" :aria-label="t('rewards.bag')">
          <button v-for="state in filters" :key="state" :aria-pressed="filter===state" @click="filter=state">{{ t(`rewards.${state==='history'?'historyCards':state}`) }}<span>{{ filterCount(state) }}</span></button>
        </nav>
        <p v-if="loading"
           role="status"
           class="rw-caption">
          {{ t('rewards.loading') }}
        </p><div v-else-if="!cards.length"
                 class="rw-empty">
          <RewardIcon name="ticket" /><h3>
            {{ t('rewards.bagWaiting') }}
          </h3><p>
            {{ t('rewards.empty') }}
          </p><RouterLink class="btn btn-primary"
                          to="/incentives">
            {{ t('rewards.benefits') }}
          </RouterLink><RouterLink class="btn btn-secondary"
                                   to="/farm">
            {{ t('rewards.farm') }}
          </RouterLink>
        </div>
        <p v-if="!loading && cards.length && !filteredCards.length" class="rw-empty">{{ t('rewards.noFilteredCards') }}</p>
        <div v-if="loading && !cards.length" class="cards-skeleton" aria-hidden="true"><span/><span/><span/></div>
        <div class="cards-grid">
          <article v-for="card in filteredCards"
                   :key="card.id"
                   class="reward-ticket"
                   :class="[`ticket-${card.template.kind}`,{'ticket-ended':!['unused','active'].includes(card.status)}]">
            <div class="ticket-top">
              <span class="ticket-symbol">
                <RewardIcon :name="cardIcon(card)" />
              </span><span class="ticket-status"
                           :class="`status-${card.status}`">
                <i />{{ t(`rewards.${card.status}`) }}
              </span>
            </div><p class="rw-eyebrow">
              {{ t(`rewards.${card.template.kind}`) }}
            </p><h2>
              {{ card.template.name }}
            </h2><p class="ticket-scope">
              {{ card.template.kind !== 'group_rate' ? templateModels(card.template).join(', ') : `${t('rewards.groups')} #${card.template.group_ids.join(', #')}` }}
            </p><div class="ticket-value">
              <template v-if="card.template.kind==='model_tokens'">
                <strong>
                  {{ card.remaining_tokens.toLocaleString() }}
                </strong><span>
                  / {{ card.template.tokens.toLocaleString() }} Token
                </span><div class="rw-progress">
                  <span :style="{width:`${tokenPercent(card)}%`}" />
                </div>
              </template><template v-else-if="card.template.kind==='group_rate'">
                <strong>
                  {{ card.template.rate }}<small>
                    ×
                  </small>
                </strong><span>
                  {{ t('rewards.rate') }} · {{ card.template.hours }} {{ t('rewards.hours') }}
                </span>
              </template><template v-else>
                <strong>
                  {{ card.template.hours }}<small>
                    h
                  </small>
                </strong><span>
                  {{ t('rewards.freeText') }}
                </span>
              </template>
            </div><div class="ticket-perforation" /><div class="ticket-bottom">
              <div class="ticket-meta">
                <span>
                  {{ t(`rewards.source_${card.source}`) }}
                </span><span>
                  #{{ card.id.toString().padStart(5,'0') }}
                </span>
              </div><p class="ticket-expiry">
                <RewardIcon name="clock" /><span>
                  {{ t(card.expires_at?'rewards.expires':'rewards.claim') }}<time>
                    {{ date(card.expires_at || card.claim_expires_at) }}
                  </time>
                </span>
              </p><button v-if="card.status==='unused'"
                          class="btn btn-primary"
                          :disabled="busy"
                          @click="pendingCard=card">
                {{ t('rewards.activate') }}<RewardIcon class="rw-button-icon"
                                                       name="arrow" />
              </button><p v-else-if="card.status==='active'"
                          class="ticket-active-note">
                <RewardIcon name="check" />{{ t('rewards.keyReady') }}
              </p><p v-else
                     class="rw-caption">
                {{ t(`rewards.${card.status}`) }}
              </p>
            </div>
          </article>
        </div>
        <p v-if="cards.length"
           class="rw-note mt-6">
          <RewardIcon name="shield" />{{ t('rewards.typeNote') }}
        </p>
      </template>
      <section v-else
               class="rw-panel cards-records">
        <div class="rw-section-title">
          <div>
            <h2>
              {{ t('rewards.records') }}
            </h2><p class="rw-caption">
              {{ t('rewards.reconciliation') }}
            </p>
          </div><RewardIcon name="shield" />
        </div><p v-if="!records.length"
                 class="rw-empty">
          {{ t('rewards.noRecords') }}
        </p><article v-for="row in records"
                     :key="row.id"
                     class="usage-row">
          <div class="usage-heading">
            <strong>
              {{ row.model }}
            </strong><time>
              {{ date(row.created_at) }}
            </time>
          </div><p class="usage-request">
            {{ t('rewards.request') }} · {{ row.request_id }}
          </p><dl class="usage-amounts">
            <div>
              <dt>
                {{ t('rewards.original') }}
              </dt><dd>
                ${{ money(row.original_cost) }}
              </dd>
            </div><span aria-hidden="true">
              −
            </span><div>
              <dt>
                {{ t('rewards.discount') }}
              </dt><dd class="discount-amount">
                ${{ money(row.discount) }}
              </dd>
            </div><span aria-hidden="true">
              =
            </span><div>
              <dt>
                {{ t('rewards.paid') }}
              </dt><dd>
                ${{ money(row.paid_cost) }}
              </dd>
            </div>
          </dl><div class="usage-tokens">
            <span>
              Token
            </span><div class="rw-progress">
              <span :style="{width:`${row.tokens>0?Math.min(100,row.covered_tokens/row.tokens*100):0}%`}" />
            </div><span>
              {{ row.covered_tokens.toLocaleString() }} / {{ row.tokens.toLocaleString() }}
            </span>
          </div>
        </article>
      </section>
      <ConfirmDialog :show="!!pendingCard"
                     :title="t('rewards.activate')"
                     :message="`${pendingCard?.template.name || ''} — ${t('rewards.activateConfirm')}`"
                     @cancel="pendingCard=undefined"
                     @confirm="confirmActivate">
        <section v-if="pendingCard" class="card-activation-preview">
          <p>{{ t('rewards.activationPreview') }}</p><h3>{{ pendingCard.template.name }}</h3>
          <RewardTerms :template="pendingCard.template" :remaining-tokens="pendingCard.remaining_tokens" /><p>{{ t('rewards.typeNote') }}</p>
        </section>
      </ConfirmDialog>
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import { computed,onMounted,ref,watch } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import RewardsNav from '@/components/rewards/RewardsNav.vue'
import RewardIcon from '@/components/rewards/RewardIcon.vue'
import RewardTerms from '@/components/rewards/RewardTerms.vue'
import RedeemView from './RedeemView.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { rewardsAPI,rewardError,type RewardCard,type RewardUsage } from '@/api/rewards'
const {t}=useI18n(),route=useRoute(),router=useRouter()
const tabs=['bag','redeem','records'] as const
type Tab=typeof tabs[number]
const tab=ref<Tab>(route.query.tab==='redeem'?'redeem':route.query.tab==='records'?'records':'bag')
const cards=ref<RewardCard[]>([]),records=ref<RewardUsage[]>([]),loading=ref(true),busy=ref(false),error=ref(''),notice=ref('')
const unusedCount=computed(()=>cards.value.filter(c=>c.status==='unused').length)
const activeCount=computed(()=>cards.value.filter(c=>c.status==='active').length)
const filters=['all','active','unused','history'] as const
type Filter=typeof filters[number]
const filter=ref<Filter>('all')
const matchesFilter=(c:RewardCard,state:Filter)=>state==='all'||(state==='history'?!['unused','active'].includes(c.status):c.status===state)
const filterCount=(state:Filter)=>cards.value.filter(c=>matchesFilter(c,state)).length
const filteredCards=computed(()=>cards.value.filter(c=>matchesFilter(c,filter.value)).sort((a,b)=>{const rank=(c:RewardCard)=>c.status==='active'?0:c.status==='unused'?1:2;return rank(a)-rank(b)||b.id-a.id}))
watch(()=>route.query.tab,(v)=>{tab.value=v==='redeem'?'redeem':v==='records'?'records':'bag'})
const availableTokens=computed(()=>cards.value.filter(c=>['unused','active'].includes(c.status)&&c.template.kind==='model_tokens').reduce((n,c)=>n+c.remaining_tokens,0))
const pendingCard=ref<RewardCard>()
const cardIcon=(c:RewardCard)=>c.template.kind==='model_tokens'?'crystal':c.template.kind==='group_rate'?'spark':'sun'
const tokenPercent=(c:RewardCard)=>c.template.tokens>0?Math.max(0,Math.min(100,c.remaining_tokens/c.template.tokens*100)):0
async function confirmActivate(){const card=pendingCard.value;pendingCard.value=undefined;if(card)await activate(card)}
const date=(v?:string)=>v?new Date(v).toLocaleString():'—',money=(v:number)=>Number(v).toFixed(8)
function select(v:Tab){tab.value=v;void router.replace({query:{...route.query,tab:v}});if(v!=='redeem')void load()}
async function load(){error.value='';loading.value=true;try{[cards.value,records.value]=await Promise.all([rewardsAPI.cards(),rewardsAPI.records()])}catch(e){error.value=rewardError(e)}finally{loading.value=false}}
async function activate(c:RewardCard){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await rewardsAPI.activate(c.id);notice.value=t('rewards.activatedCard',{name:c.template.name});await load()}catch(e){error.value=rewardError(e)}finally{busy.value=false}}
onMounted(load)
</script>
