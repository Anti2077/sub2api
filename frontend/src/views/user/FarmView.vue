<template>
  <AppLayout>
    <FarmSprites /><div class="reward-page farm-page">
      <RewardsNav current="farm" />
      <header class="rw-header">
        <div>
          <p class="rw-eyebrow">
            TOKEN GARDEN / 03
          </p><h1>
            {{ t('rewards.home') }}<span class="rw-title-mark">
              .
            </span>
          </h1><p>
            {{ t('rewards.farmNote') }}
          </p>
        </div><div class="farm-wallet">
          <RewardIcon name="crystal" /><div>
            <small>
              {{ t('rewards.crystals') }}
            </small><strong>
              {{ crystals.toLocaleString() }}
            </strong>
          </div><button class="rw-link"
                        @click="changeTab('ledger')">
            {{ t('rewards.ledger') }}<RewardIcon name="arrow" />
          </button>
        </div>
      </header>
      <div class="rw-flow">
        <span>
          <b>
            01
          </b>{{ t('rewards.plant') }}
        </span><i>
          →
        </i><span>
          <b>
            02
          </b>{{ t('rewards.harvest') }}
        </span><i>
          →
        </i><span>
          <b>
            03
          </b>{{ t('rewards.buy') }}
        </span><small>
          {{ t('rewards.offlineGrowth') }}
        </small>
      </div>
      <div class="farm-toolbar">
        <nav class="rw-segments"
             :aria-label="t('rewards.farm')">
          <button v-for="v in tabs"
                  :key="v"
                  :aria-pressed="tab===v"
                  @click="changeTab(v)">
            {{ t(`rewards.${v}`) }}<span v-if="v==='home'&&readyCount">
              {{ readyCount }}
            </span>
          </button>
        </nav><button class="rw-link"
                      :disabled="busy||loading"
                      @click="load">
          <RewardIcon name="refresh" />{{ t('rewards.refresh') }}
        </button>
      </div>
      <div class="farm-messages" aria-live="polite"><p v-if="error"
         role="alert"
         class="rw-alert">
        {{ error }}
      </p><p v-if="notice"
             role="status"
             class="rw-success">
        <RewardIcon name="check" />{{ notice }}
      </p></div><p v-if="loading && !farm"
             role="status"
             class="rw-caption">
        {{ t('rewards.loading') }}
      </p>
      <div v-if="loading && !farm && tab==='home'"
           class="farm-loading-skeleton"
           aria-hidden="true">
        <span/><span/><span/><span/><span/><span/>
      </div>
      <template v-if="tab==='home' && farm">
        <div class="farm-layout">
          <section class="farm-scene"
                   :aria-label="t('rewards.greenhouse')">
            <div class="farm-world"><FarmLandscape :plots="farm.plots.length" /><div class="farm-pods">
              <button v-for="(p,index) in farm.plots"
                      :key="p.slot"
                      :style="fieldPosition(index)"
                      class="farm-plot"
                      :class="{'farm-ripe':ripe(p),'farm-vacant':!p.crop||p.harvested_at,'farm-selected':selectedSlot===p.slot}"
                      :aria-disabled="busy"
                      :aria-busy="busy && selectedSlot===p.slot"
                      :aria-pressed="selectedSlot===p.slot"
                      :aria-label="`#${p.slot} ${(p.crop&&!p.harvested_at ? p.crop.name : t('rewards.emptyPlot'))} ${plotLabel(p)}`"
                      @click="inspectOrAct(p)">
                <span class="farm-hit"
                      aria-hidden="true"/><span class="pod-number">
                        {{ p.slot.toString().padStart(2,'0') }}
                      </span><span v-if="ripe(p)"
                             class="pod-ready">
                  <RewardIcon :name="owner&&protectedPlot(p)?'shield':'spark'" />{{ t(owner&&protectedPlot(p)?'rewards.protected':'rewards.ready') }}
                </span><div class="pod-bed">
                  <FarmSoil /><svg class="farm-selection" viewBox="0 0 200 100" aria-hidden="true"><path d="M100 2 196 50 100 98 4 50Z" /></svg><span v-if="feedback?.slot===p.slot"
                                    class="pod-feedback">
                    <RewardIcon :name="feedback.kind==='plant'?'leaf':'crystal'" />{{ feedback.text }}
                  </span><FarmCrop v-if="p.crop&&!p.harvested_at" class="farm-crop" field :class="{'is-ripe':ripe(p),'is-young':growth(p)<35}" :kind="p.crop.id" :young="growth(p)<35"><svg
                              viewBox="0 0 180 120"
                              class="crop-art"
                              :class="{'is-ripe':ripe(p),'is-young':growth(p)<35}"
                              aria-hidden="true">
                    <use :href="sprite(p)"
                         x="0"
                         y="21"
                         width="68"
                         height="90"/><use :href="sprite(p)"
                                           x="57"
                                           y="0"
                                           width="72"
                                           height="95"/><use :href="sprite(p)"
                                                             x="111"
                                                             y="26"
                                                             width="68"
                                                             height="90"/>
                  </svg></FarmCrop><RewardIcon v-else
                                    class="pod-plus"
                                    name="plus" />
                </div><strong>
                  {{ p.crop&&!p.harvested_at?p.crop.name:t('rewards.emptyPlot') }}
                </strong><span class="pod-label">
                  {{ plotLabel(p) }}
                </span><div v-if="p.crop&&!p.harvested_at"
                            class="pod-progress">
                  <span :style="{width:`${growth(p)}%`}" />
                </div><span v-if="p.crop&&!p.harvested_at"
                            class="pod-yield">
                  <RewardIcon name="crystal" />{{ p.crop.yield-p.stolen }}<small v-if="p.stolen">
                    · {{ t('rewards.used') }} {{ p.stolen }}
                  </small>
                </span>
              </button>
            </div></div><div class="farm-scene-top">
              <span class="farm-scene-tag">
                <RewardIcon name="sun" />{{ owner ? `#${owner} · ${t('rewards.farm')}` : t('rewards.greenhouse') }}
              </span><button v-if="owner"
                             class="btn btn-secondary"
                             @click="visit(0)">
                {{ t('rewards.back') }}
              </button><span v-else
                             class="farm-live">
                <i />{{ t('rewards.growingQuietly') }}
              </span>
            </div>
            <div class="farm-scene-footer">
              <RewardIcon name="leaf" /><span v-if="selectedPlot"
                                              class="farm-selected-detail">
                <b>
                  #{{ selectedPlot.slot }} · {{ selectedPlot.crop&&!selectedPlot.harvested_at?selectedPlot.crop.name:t('rewards.emptyPlot') }}
                </b><small>
                  {{ plotLabel(selectedPlot) }}<template v-if="selectedPlot.crop&&!selectedPlot.harvested_at">
                    · {{ t('rewards.yield') }} {{ selectedPlot.crop.yield-selectedPlot.stolen }}
                  </template>
                </small>
              </span><span v-else>
                {{ t('rewards.sceneNote') }}
              </span><span>
                {{ farm.plots.length.toString().padStart(2,'0') }} / {{ t('rewards.plotsShort') }}
              </span>
            </div>
          </section>
          <aside class="farm-aside">
            <section v-if="selectedPlot?.crop && !selectedPlot.harvested_at" class="rw-panel farm-crop-detail" aria-live="polite">
              <div class="rw-section-title"><h2>#{{ selectedPlot.slot }} · {{ selectedPlot.crop.name }}</h2><RewardIcon :name="ripe(selectedPlot)?'basket':'leaf'" /></div>
              <FarmCrop :kind="selectedPlot.crop.id" :young="growth(selectedPlot)<35" aria-hidden="true" />
              <strong>{{ plotLabel(selectedPlot) }}</strong>
              <div class="rw-progress" role="progressbar" :aria-label="t('rewards.growing')" :aria-valuenow="Math.floor(growth(selectedPlot))" aria-valuemin="0" aria-valuemax="100"><span :style="{width:`${growth(selectedPlot)}%`}" /></div>
              <dl class="reward-terms"><div><dt>{{ t('rewards.yield') }}</dt><dd>{{ selectedPlot.crop.yield-selectedPlot.stolen }} {{ t('rewards.crystals') }}</dd></div><div><dt>{{ t('rewards.maturesAt') }}</dt><dd>{{ new Date(selectedPlot.ready_at!).toLocaleString() }}</dd></div></dl>
              <button v-if="canAct(selectedPlot)" class="btn btn-primary" :disabled="busy" @click="plotAction(selectedPlot)">{{ t(owner?'rewards.steal':'rewards.harvest') }}</button>
              <button v-if="!owner" class="rw-link" @click="selectedSlot=undefined">{{ t('rewards.chooseAnotherSeed') }}<RewardIcon name="arrow" /></button>
            </section>
            <section v-else-if="selectedSeed && !owner" class="rw-panel farm-action-guide">
              <div class="rw-section-title"><h2>{{ t('rewards.readyToPlant') }}</h2><RewardIcon name="leaf" /></div>
              <FarmCrop :kind="selectedSeed.id" aria-hidden="true" />
              <strong>{{ selectedSeed.name }}</strong>
              <p>{{ selectedSeed.hours }} {{ t('rewards.hours') }} · +{{ selectedSeed.yield }} {{ t('rewards.crystals') }}</p>
              <small>{{ t('rewards.plantGuide') }}</small>
            </section>
            <section v-if="nextGoal"
                     class="rw-panel farm-goal">
              <div class="rw-section-title">
                <h2>
                  {{ t('rewards.nextCard') }}
                </h2><RewardIcon name="ticket" />
              </div><span class="rw-caption">
                {{ t(`rewards.${nextGoal.template.kind}`) }}
              </span><h3>
                {{ nextGoal.template.name }}
              </h3><div class="rw-progress">
                <span :style="{width:`${nextGoal.price>0?Math.min(100,crystals/nextGoal.price*100):100}%`}" />
              </div><p class="goal-count">
                {{ crystals.toLocaleString() }} <span>
                  / {{ nextGoal.price.toLocaleString() }} {{ t('rewards.crystals') }}
                </span>
              </p><button class="rw-link"
                          @click="changeTab('shop')">
                {{ t('rewards.shop') }}<RewardIcon name="arrow" />
              </button>
            </section>
            <section v-if="!owner" class="rw-panel farm-seeds">
              <div class="rw-section-title">
                <h2>
                  {{ t('rewards.seed') }}
                </h2><RewardIcon name="leaf" />
              </div><label v-for="c in farm.config.crops.filter(c=>c.enabled)"
                           :key="c.id"
                           class="seed-option"
                           :class="{'is-selected':seed===c.id}">
                <input v-model="seed"
                       type="radio"
                       name="seed"
                       :value="c.id"><FarmCrop class="crop-mini" :kind="c.id"><svg
                                          aria-hidden="true">
                         <use :href="cropSprite(c.id)" />
                       </svg></FarmCrop><span>
                  <strong>
                    {{ c.name }}
                  </strong><small>
                    {{ c.hours }} {{ t('rewards.hours') }} · +{{ c.yield }}
                  </small>
                </span><RewardIcon v-if="seed===c.id"
                                   name="check" />
              </label><p class="rw-caption">
                {{ t('rewards.seedHint') }}
              </p>
            </section>
          </aside>
        </div><section class="rw-panel farm-bottom">
          <div class="rw-note">
            <RewardIcon name="shield" /><p>
              {{ t('rewards.rules') }}
            </p>
          </div><label v-if="!owner"
                       class="farm-privacy">
            <input type="checkbox"
                   :checked="farm.public"
                   :disabled="busy"
                   @change="privacy(($event.target as HTMLInputElement).checked)"><span>
                     <strong>
                       {{ t('rewards.public') }}
                     </strong><small>
                       {{ t('rewards.privateNote') }}
                     </small>
                   </span>
          </label>
        </section>
      </template>
      <section v-if="tab==='neighbors'"
               class="rw-panel farm-directory">
        <div class="rw-section-title">
          <div>
            <h2>
              {{ t('rewards.neighbors') }}
            </h2><p class="rw-caption">
              {{ t('rewards.neighborNote') }}
            </p>
          </div><RewardIcon name="friends" />
        </div><form class="flex gap-2"
                    @submit.prevent="search">
          <input v-model="query"
                 class="input min-w-0 flex-1"
                 :aria-label="t('rewards.search')"
                 :placeholder="t('rewards.search')"><button class="btn btn-primary"
                                                            :disabled="busy">
                   {{ t('rewards.search') }}
                 </button>
        </form><p v-if="!neighbors.length"
                  class="rw-empty">
          {{ t('rewards.noNeighbors') }}
        </p><div class="neighbor-grid">
          <article v-for="n in neighbors"
                   :key="n.user_id"
                   class="neighbor-card">
            <svg class="neighbor-avatar"
                 aria-hidden="true">
              <use href="#farmer" />
            </svg><div>
              <strong>
                {{ n.nickname }}
              </strong><small>
                #{{ n.user_id }}
              </small>
            </div><button class="rw-link"
                          :disabled="busy"
                          @click="visit(n.user_id)">
              {{ t('rewards.visit') }}<RewardIcon name="arrow" />
            </button>
          </article>
        </div>
      </section>
      <section v-if="tab==='shop'"
               class="farm-shop-grid">
        <p v-if="purchaseDone" class="rw-success shop-receipt" role="status"><RewardIcon name="check" /><span>{{ t('rewards.notice') }}</span><RouterLink class="rw-link" to="/cards?tab=bag">{{ t('rewards.viewBag') }}<RewardIcon name="arrow" /></RouterLink></p>
        <article v-for="s in shop"
                 :key="s.id"
                 class="rw-panel farm-shop-card">
          <div class="shop-ticket-icon">
            <RewardIcon name="ticket" />
          </div><p class="rw-eyebrow">
            {{ t(`rewards.${s.template.kind}`) }}
          </p><h2>
            {{ s.template.name }}
          </h2><p class="rw-caption break-all">
            {{ s.template.kind !== 'group_rate' ? templateModels(s.template).join(', ') : `${t('rewards.groups')} ${s.template.group_ids.join(', ')}` }}
          </p><div class="shop-price">
            <RewardIcon name="crystal" /><strong>
              {{ s.price.toLocaleString() }}
            </strong><span>
              {{ t('rewards.crystals') }}
            </span>
          </div><RewardTerms :template="s.template" /><p class="rw-caption">
            {{ t('rewards.stock') }} {{ s.stock }} · {{ t('rewards.daily') }} {{ s.daily_limit }} · {{ t('rewards.total') }} {{ s.total_limit }}
          </p><button class="btn btn-primary"
                      :disabled="busy || crystals<s.price || s.stock===0"
                      @click="pendingShop=s">
            {{ t('rewards.buy') }}<RewardIcon class="rw-button-icon"
                                              name="arrow" />
          </button>
        </article><p v-if="!shop.length"
                     class="rw-empty">
          {{ t('rewards.noRecords') }}
        </p>
      </section>
      <section v-if="tab==='ledger'"
               class="rw-panel farm-ledger">
        <div class="rw-section-title">
          <h2>
            {{ t('rewards.ledger') }}
          </h2><RewardIcon name="crystal" />
        </div><p v-if="!ledger.length"
                 class="rw-empty">
          {{ t('rewards.noRecords') }}
        </p><article v-for="l in ledger"
                     :key="l.id"
                     class="farm-ledger-row">
          <span class="ledger-icon">
            <RewardIcon :name="l.delta>0?'basket':'ticket'" />
          </span><div>
            <strong>
              {{ t(`rewards.${l.kind==='exchange'?'buy':l.kind}`) }}
            </strong><time>
              {{ new Date(l.created_at).toLocaleString() }}
            </time>
          </div><strong :class="{'is-positive':l.delta>0}">
            {{ l.delta>0?'+':'' }}{{ l.delta }}
          </strong>
        </article>
      </section>
      <ConfirmDialog :show="!!pendingShop"
                     :title="t('rewards.buy')"
                     :message="t('rewards.buyConfirm')"
                     @cancel="pendingShop=undefined"
                     @confirm="confirmBuy">
        <section v-if="pendingShop" class="card-activation-preview">
          <h3>{{ pendingShop.template.name }}</h3><RewardTerms :template="pendingShop.template" />
          <dl class="reward-terms exchange-cost"><div><dt>{{ t('rewards.price') }}</dt><dd>−{{ pendingShop.price.toLocaleString() }}</dd></div><div><dt>{{ t('rewards.afterExchange') }}</dt><dd>{{ (crystals-pendingShop.price).toLocaleString() }} {{ t('rewards.crystals') }}</dd></div></dl>
          <p>{{ t('rewards.exchangeActivationNote') }}</p>
        </section>
      </ConfirmDialog>
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { templateModels } from '@/components/rewards/templateValidation'
import {computed,onMounted,onBeforeUnmount,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import FarmSprites from '@/components/rewards/FarmSprites.vue'
import FarmLandscape from '@/components/rewards/FarmLandscape.vue'
import { farmGeometry } from '@/components/rewards/farmGeometry'
import FarmSoil from '@/components/rewards/FarmSoil.vue'
import FarmCrop from '@/components/rewards/FarmCrop.vue'
import RewardIcon from '@/components/rewards/RewardIcon.vue'
import RewardTerms from '@/components/rewards/RewardTerms.vue'
import RewardsNav from '@/components/rewards/RewardsNav.vue'
import {rewardsAPI,rewardError,type Farm,type Plot,type Neighbor,type Shop,type Ledger,type FarmAction} from '@/api/rewards'
const {t}=useI18n(),tabs=['home','neighbors','shop','ledger'] as const
type Tab=typeof tabs[number]
const tab=ref<Tab>('home'),farm=ref<Farm>(),owner=ref(0),crystals=ref(0),seed=ref('sprout'),neighbors=ref<Neighbor[]>([]),shop=ref<Shop[]>([]),ledger=ref<Ledger[]>([]),query=ref(''),busy=ref(false),loading=ref(false),error=ref(''),notice=ref(''),now=ref(Date.now())
let offset=0,timer:ReturnType<typeof setInterval>|undefined
const field = computed(() => farmGeometry(farm.value?.plots.length || 12))
const fieldPosition = (index: number) => field.value.position(index)
const readyCount=computed(()=>farm.value?.plots.filter(p=>ripe(p)).length||0)
const nextGoal=computed(()=>shop.value.filter(s=>s.stock!==0).sort((a,b)=>a.price-b.price)[0])
const selectedSeed=computed(()=>farm.value?.config.crops.find(c=>c.enabled&&c.id===seed.value))
function growth(p:Plot){if(!p.crop||p.harvested_at||!p.ready_at||!p.planted_at)return 0;return Math.min(100,Math.max(0,(now.value-Date.parse(p.planted_at))/(Date.parse(p.ready_at)-Date.parse(p.planted_at))*100))}
const ripe=(p:Plot)=>!!p.crop&&!p.harvested_at&&!!p.ready_at&&now.value>=Date.parse(p.ready_at)
function cropSprite(id:string){return ({sprout:'#crop-sprout',cluster:'#crop-tomato',bloom:'#crop-carrot',moon:'#crop-corn'} as Record<string,string>)[id]||'#crop-sprout'}
function sprite(p:Plot){return growth(p)<35?'#crop-sprout':cropSprite(p.crop!.id)}
const protectedPlot=(p:Plot)=>ripe(p)&&now.value<Date.parse(p.ready_at!)+p.crop!.protection_minutes*60000
function plotLabel(p:Plot){if(!p.crop||p.harvested_at)return t('rewards.plant');if(ripe(p))return t(owner.value?(protectedPlot(p)?'rewards.protected':'rewards.steal'):'rewards.harvest');const m=Math.max(1,Math.ceil((Date.parse(p.ready_at!)-now.value)/60000));return `${t('rewards.growing')} · ${Math.floor(m/60)}h ${m%60}m`}
function canAct(p:Plot){return owner.value?ripe(p)&&!protectedPlot(p):!p.crop||!!p.harvested_at||ripe(p)}
async function load(){loading.value=true;error.value='';try{farm.value=await rewardsAPI.farm(owner.value);offset=Date.parse(farm.value.server_time)-Date.now();now.value=Date.now()+offset;if(!owner.value)crystals.value=farm.value.crystals;shop.value=await rewardsAPI.shop();ledger.value=await rewardsAPI.ledger()}catch(e){error.value=rewardError(e)}finally{loading.value=false}}
// Reuse a key after an uncertain network/server failure so retrying an exchange
// cannot create a second order. A later planting cycle is a distinct operation.
const pendingActions=new Map<string,string>()
async function action(v:Omit<FarmAction,'request_key'>){
 if(busy.value)return false
 const identity=JSON.stringify([v,v.slot?farm.value?.plots.find(p=>p.slot===v.slot)?.cycle:null,v.shop_id?shop.value.find(s=>s.id===v.shop_id)?.version:null])
 const key=pendingActions.get(identity)||crypto.randomUUID()
 pendingActions.set(identity,key)
 busy.value=true;error.value='';notice.value=''
 try{await rewardsAPI.action({...v,request_key:key});pendingActions.delete(identity);notice.value=t('rewards.success');await load();return true}
 catch(e){const status=(e as {response?:{status?:number}}).response?.status;if(status&&status>=400&&status<500&&status!==408&&status!==429)pendingActions.delete(identity);error.value=rewardError(e);return false}
 finally{busy.value=false}
}
const feedback=ref<{slot:number;kind:string;text:string}>()
let feedbackTimer:ReturnType<typeof setTimeout>|undefined
const selectedSlot=ref<number>()
const selectedPlot=computed(()=>farm.value?.plots.find(p=>p.slot===selectedSlot.value))
async function inspectOrAct(p:Plot){
 if(busy.value)return
 selectedSlot.value=p.slot
 // Growing and protected crops may be inspected; all original actions still
 // use the same server-backed handler once their existing gate is satisfied.
 if(canAct(p))await plotAction(p)
}
async function plotAction(p:Plot){if(busy.value)return;const kind=owner.value?'steal':ripe(p)?'harvest':'plant';const before=crystals.value;const success=owner.value?await action({kind:'steal',owner_id:owner.value,slot:p.slot}):await action({kind:ripe(p)?'harvest':'plant',slot:p.slot,crop_id:seed.value});if(success&&!error.value){feedback.value={slot:p.slot,kind,text:kind==='plant'?t('rewards.planted'):`+${crystals.value-before}`};clearTimeout(feedbackTimer);feedbackTimer=setTimeout(()=>{feedback.value=undefined},2200)}}
async function privacy(v:boolean){await action({kind:'privacy',public:v})}
const pendingShop=ref<Shop>(),purchaseDone=ref(false)
async function confirmBuy(){const s=pendingShop.value;pendingShop.value=undefined;if(!s||busy.value)return;purchaseDone.value=false;purchaseDone.value=await action({kind:'buy',shop_id:s.id})}
async function search(){error.value='';try{neighbors.value=await rewardsAPI.neighbors(query.value)}catch(e){error.value=rewardError(e)}}
async function visit(id:number){owner.value=id;selectedSlot.value=undefined;tab.value='home';await load()}
function changeTab(v:Tab){tab.value=v;if(v==='neighbors')void search();else if(v==='home'&&owner.value)void visit(0);else void load()}
onMounted(()=>{void load();timer=setInterval(()=>{now.value=Date.now()+offset},1000)})
onBeforeUnmount(()=>{clearInterval(timer);clearTimeout(feedbackTimer)})
</script>
