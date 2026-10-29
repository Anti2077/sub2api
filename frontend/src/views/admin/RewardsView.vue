<template>
<AppLayout><div class="mx-auto max-w-6xl space-y-6"><header><h1 class="text-2xl font-semibold">{{ t(farmMode?'rewards.farmAdmin':'rewards.admin') }}</h1><p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ t(farmMode?'rewards.farmConfigNote':'rewards.typeNote') }}</p></header>
<nav class="flex flex-wrap gap-2"><button v-for="v in tabs" :key="v" class="btn min-h-11" :class="tab===v?'btn-primary':'btn-secondary'" :disabled="busy" @click="switchTab(v)">{{ t(`rewards.${v}`) }}</button><button class="btn btn-secondary ml-auto" :disabled="busy" @click="load">{{ t('rewards.refresh') }}</button></nav>
<p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p><p v-if="notice" role="status" class="text-emerald-700 dark:text-emerald-300">{{ notice }}</p>
<section v-if="tab==='templates'" class="space-y-4">
<form v-if="editor" class="card space-y-4 p-6" @submit.prevent="saveTemplate">
<fieldset :disabled="busy" class="space-y-4"><TemplateFields ref="templateFields" :key="editor.id" v-model="editor" :groups="groups" :disabled="busy" />
<div class="flex gap-2"><button class="btn btn-primary">{{ t('rewards.save') }}</button><button type="button" class="btn btn-secondary" @click="editor=undefined">{{ t('rewards.cancel') }}</button></div></fieldset></form>
<TemplateManager :templates="templates" :groups="groups" :busy="busy || !!editor" :reload="reloadData" @edit="editTemplate" @busy="busy=$event" />
</section>
<CardGrantManager v-if="tab==='grants'" :cards="cards" :templates="templates" :groups="groups" :busy="busy" :reload="reloadData" @busy="busy=$event" />
<CodeManager v-if="tab==='codes'" :templates="templates" :groups="groups" :busy="busy" @busy="busy=$event" />
<form v-if="tab==='config' && config" class="card space-y-6 p-6" @submit.prevent="saveConfig"><fieldset :disabled="busy" class="space-y-6"><div class="grid gap-3 sm:grid-cols-2"><label v-for="k in boolKeys" :key="k" class="flex min-h-11 items-center gap-3"><input v-model="config[k]" type="checkbox">{{ t(`rewards.${k}`) }}</label></div><div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><label v-for="k in numKeys" :key="k">{{ t(`rewards.${k}`) }}<input v-model.number="config[k]" class="input mt-1" type="number" :min="k==='plots'?1:0" :max="k==='plots'?16:undefined" required></label><label>{{ t('rewards.starts') }}<input v-model="start" class="input mt-1" placeholder="2026-10-20T00:00:00Z"></label><label>{{ t('rewards.ends') }}<input v-model="end" class="input mt-1" placeholder="2026-11-20T00:00:00Z"></label><label>{{ t('rewards.allowed') }}<input v-model="allowed" class="input mt-1"></label></div><RecipientPicker :model-value="allowedIDs" :disabled="busy" :title="t('rewards.manageChooseAudience')" :max-selected="10000" @update:model-value="allowed=$event.join(',')" /><CropManager v-model="config.crops" :disabled="busy" /><button class="btn btn-primary" :disabled="busy">{{ t('rewards.save') }}</button></fieldset></form>
<section v-if="tab==='shop'" class="space-y-4"><form v-if="shopEditor" class="card space-y-4 p-6" @submit.prevent="saveShop"><fieldset :disabled="busy" class="space-y-4"><TemplatePicker v-model="shopEditor.template_id" :templates="templates" :groups="groups" /><div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><label>{{ t('rewards.price') }}<input v-model.number="shopEditor.price" class="input mt-1" type="number" min="1" required></label><label>{{ t('rewards.stock') }}<input v-model.number="shopEditor.stock" class="input mt-1" type="number" min="0" required></label><label>{{ t('rewards.daily') }}<input v-model.number="shopEditor.daily_limit" class="input mt-1" type="number" min="1" required></label><label>{{ t('rewards.total') }}<input v-model.number="shopEditor.total_limit" class="input mt-1" type="number" :min="shopEditor.daily_limit" required></label><label>{{ t('rewards.starts') }}<input v-model="shopStart" class="input mt-1" type="datetime-local"></label><label>{{ t('rewards.ends') }}<input v-model="shopEnd" class="input mt-1" type="datetime-local"></label></div><label class="flex min-h-11 items-center gap-2"><input v-model="shopEditor.published" type="checkbox">{{ t('rewards.published') }}</label><div class="flex gap-2"><button class="btn btn-primary min-h-11">{{ t('rewards.save') }}</button><button type="button" class="btn btn-secondary min-h-11" @click="shopEditor=undefined">{{ t('rewards.cancel') }}</button></div></fieldset></form><ShopManager :items="shop" :busy="busy || !!shopEditor" :reload="reloadData" @edit="editShop" @busy="busy=$event" /></section>
<RecordsManager v-if="tab==='audit'" :items="audit" :title="t('rewards.audit')" />
<RecordsManager v-if="tab==='ledger'" :items="ledger" :title="t('rewards.ledger')" />
</div></AppLayout>
</template>
<script setup lang="ts">
import {computed,onMounted,onBeforeUnmount,ref,watch} from 'vue'
import {onBeforeRouteLeave,onBeforeRouteUpdate,useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TemplateFields from '@/components/rewards/TemplateFields.vue'
import {validModelIDs,templateModels} from '@/components/rewards/templateValidation'
import RecipientPicker from '@/components/rewards/RecipientPicker.vue'
import TemplateManager from '@/components/rewards/TemplateManager.vue'
import {invalidCrops} from '@/components/rewards/managementValidation'
import CardGrantManager from '@/components/rewards/CardGrantManager.vue'
import CodeManager from '@/components/rewards/CodeManager.vue'
import ShopManager from '@/components/rewards/ShopManager.vue'
import CropManager from '@/components/rewards/CropManager.vue'
import RecordsManager from '@/components/rewards/RecordsManager.vue'
import TemplatePicker from '@/components/rewards/TemplatePicker.vue'
import {adminAPI} from '@/api/admin'
import type {AdminGroup} from '@/types'
import {rewardsAPI,rewardError,type RewardConfig,type CardTemplate,type RewardCard,type Shop,type Audit,type Ledger} from '@/api/rewards'
const {t}=useI18n(),route=useRoute(),farmMode=computed(()=>route.path==='/admin/farm')
const tabs=computed(()=>farmMode.value?['config','shop','ledger','audit']:['templates','grants','codes','config','audit'])
const tab=ref(farmMode.value?'config':'templates'),busy=ref(false),error=ref(''),notice=ref(''),templates=ref<CardTemplate[]>([]),cards=ref<RewardCard[]>([]),config=ref<RewardConfig>(),shop=ref<Shop[]>([]),audit=ref<Audit[]>([]),ledger=ref<Ledger[]>([]),groups=ref<AdminGroup[]>([]),editor=ref<CardTemplate>(),shopEditor=ref<Shop>(),chosen=ref(0),start=ref(''),end=ref(''),allowed=ref(''),shopStart=ref(''),shopEnd=ref('')
const templateFields=ref<InstanceType<typeof TemplateFields>>()
const configBaseline=ref('')
const allowedIDs=computed(()=>[...new Set(allowed.value.split(',').map(v=>Number(v.trim())).filter(v=>Number.isSafeInteger(v)&&v>0))])
const configState=()=>JSON.stringify([config.value,start.value,end.value,allowed.value])
const hasDraft=()=>!!editor.value||!!shopEditor.value||!!configBaseline.value&&configState()!==configBaseline.value
function allowDiscard(){return !hasDraft()||window.confirm(t('rewards.manageDiscardDraft'))}
function switchTab(value:string){if(busy.value||!allowDiscard())return;editor.value=undefined;shopEditor.value=undefined;if(configBaseline.value){[config.value,start.value,end.value,allowed.value]=JSON.parse(configBaseline.value)}tab.value=value}
function beforeUnload(e:BeforeUnloadEvent){if(hasDraft()||busy.value){e.preventDefault();e.returnValue=''}}
onBeforeRouteLeave(()=>!busy.value&&allowDiscard())
onBeforeRouteUpdate(()=>{if(busy.value||!allowDiscard())return false;editor.value=undefined;shopEditor.value=undefined;if(configBaseline.value){[config.value,start.value,end.value,allowed.value]=JSON.parse(configBaseline.value)}return true})
window.addEventListener('beforeunload',beforeUnload)
onBeforeUnmount(()=>window.removeEventListener('beforeunload',beforeUnload))
const boolKeys=['enabled','activation_enabled','farm_enabled','planting_enabled','exchange_enabled','stealing_enabled','maintenance'] as const
const numKeys=['plots','protection_minutes','steal_percent','max_stolen_percent','daily_steals','daily_per_owner'] as const
watch(farmMode,()=>{tab.value=farmMode.value?'config':'templates'})
async function reloadData(){[templates.value,cards.value,config.value,shop.value,audit.value,ledger.value,groups.value]=await Promise.all([rewardsAPI.templates(),rewardsAPI.cards(true),rewardsAPI.config(),rewardsAPI.shop(true),rewardsAPI.audit(),rewardsAPI.ledger(true),adminAPI.groups.getAll()]);start.value=config.value.starts_at||'';end.value=config.value.ends_at||'';allowed.value=(config.value.allowed_user_ids||[]).join(',');configBaseline.value=configState();if(!templates.value.some(v=>v.id===chosen.value&&v.published))chosen.value=templates.value.find(v=>v.published)?.id||0}
async function load(){if(busy.value||!allowDiscard())return;editor.value=undefined;shopEditor.value=undefined;busy.value=true;error.value='';try{await reloadData()}catch(e){error.value=rewardError(e)}finally{busy.value=false}}
async function run(fn:()=>Promise<unknown>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn();notice.value=t('rewards.success');await reloadData()}catch(e){error.value=rewardError(e)}finally{busy.value=false}}
function editTemplate(v?:CardTemplate){editor.value=v?JSON.parse(JSON.stringify(v)):{id:0,version:0,name:'',kind:'model_day',model:'',models:[],group_ids:[],rate:0.35,tokens:100000,hours:24,claim_days:30,published:false}}
function saveTemplate(){if(!templateFields.value?.prepare())return;if(editor.value&&editor.value.kind!=='group_rate'&&!validModelIDs(templateModels(editor.value))){error.value=t('rewards.invalidModels');return}if(editor.value&&editor.value.kind==='group_rate'&&editor.value.group_ids.length!==1){error.value=t('rewards.rateGroupRequired');return}if(editor.value&&!editor.value.group_ids.length){error.value=t('rewards.manageTemplateGroupRequired');return}void run(async()=>{if(!editor.value)return;await rewardsAPI.saveTemplate(editor.value);editor.value=undefined})}
function iso(v:string){if(!v.trim())return null;const d=new Date(v);if(!Number.isFinite(d.getTime()))throw new Error(t('rewards.invalidTime'));return d.toISOString()}
function saveConfig(){if(config.value&&invalidCrops(config.value.crops).length){error.value=t('rewards.manageInvalidCrops',{count:invalidCrops(config.value.crops).length});return}void run(async()=>{if(!config.value)return;config.value.starts_at=iso(start.value);config.value.ends_at=iso(end.value);config.value.allowed_user_ids=allowed.value.trim()?allowed.value.split(',').map(x=>Number(x.trim())):[];await rewardsAPI.saveConfig(config.value)})}
function localTime(v?:string|null){if(!v)return '';const d=new Date(v);return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,16)}
function editShop(v?:Shop){shopEditor.value=v?JSON.parse(JSON.stringify(v)):{id:0,version:0,template_id:chosen.value,template:{} as CardTemplate,price:50,stock:100,daily_limit:1,total_limit:5,published:false,starts_at:null,ends_at:null};shopStart.value=localTime(shopEditor.value?.starts_at);shopEnd.value=localTime(shopEditor.value?.ends_at)}
function saveShop(){void run(async()=>{if(!shopEditor.value)return;shopEditor.value.starts_at=iso(shopStart.value);shopEditor.value.ends_at=iso(shopEnd.value);await rewardsAPI.saveShop(shopEditor.value);shopEditor.value=undefined})}
onMounted(load)
</script>
