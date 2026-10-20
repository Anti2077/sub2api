<template>
  <div class="space-y-4" data-testid="fingerpoint-report">
    <div class="card p-4 sm:p-5">
      <div class="flex flex-wrap justify-between gap-4"><div><p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">LM Fingerpoint Detector</p><h3 class="mt-2 break-words text-xl font-semibold text-gray-900 dark:text-white">{{ prediction || t('admin.modelIdentity.awaitingSamples') }}</h3><p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.expectedComparison') }} <span class="font-mono">{{ run.expected_model }}</span></p></div><div class="text-left sm:text-right"><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.confidence') }}</p><p class="mt-1 text-2xl font-semibold tabular-nums">{{ percent(analysis?.probability) }}</p><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.closedSetHint') }}</p></div></div>
      <p class="mt-4"><span class="rounded px-2 py-1 text-xs font-medium" :class="identityStatusClass(identityRunStatus(run))">{{ t(`admin.modelIdentity.${identityRunStatus(run)}`) }}</span></p>
      <p class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.rankingCaution') }}</p>
      <p v-if="candidateDiffers" role="status" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200">{{ t('admin.modelIdentity.candidateDiffers') }}</p>
      <p v-if="isAstra" class="mt-4 rounded-lg bg-gray-50 p-3 text-sm text-gray-700 dark:bg-dark-900 dark:text-gray-300">{{ t('admin.modelIdentity.astraSharedHint') }}</p>
      <p v-if="analysis?.evidence?.reason" class="mt-4 text-sm text-gray-600 dark:text-gray-300">{{ analysis.evidence.reason }}</p>
      <div v-if="warnings.length" class="mt-4 space-y-2" role="status"><p v-for="warning in warnings" :key="warning" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200">{{ warning }}</p></div>
      <p v-if="report.error" role="alert" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ report.error }}</p>
    </div>
    <div class="grid gap-3 lg:grid-cols-3">
      <article v-for="(sample, index) in samples" :key="sample.id || index" class="card min-w-0 overflow-hidden" :data-sample="index">
        <header class="flex items-center justify-between gap-3 border-b border-gray-100 p-4 dark:border-dark-600"><h4 class="font-medium">{{ t('admin.modelIdentity.sampleNumber', { number: index + 1 }) }}</h4><span class="rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-300">{{ sample.status || sample.state }}</span></header>
        <div class="space-y-3 p-4"><details><summary class="min-h-11 cursor-pointer text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.samplePrompt') }} · {{ sample.expected_count }}</summary><p class="whitespace-pre-wrap break-words text-xs leading-relaxed">{{ sample.prompt }}</p></details>
          <div class="flex justify-between gap-2 text-xs text-gray-500 dark:text-gray-400"><span>{{ t('admin.modelIdentity.validNumbers') }}</span><span class="tabular-nums">{{ sample.valid_numbers ?? 0 }} / {{ sample.expected_count }}</span></div>
          <pre class="h-44 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 font-mono text-xs leading-relaxed text-gray-800 dark:bg-dark-900 dark:text-gray-200">{{ sample.text || '—' }}</pre>
          <p v-if="sample.error" role="alert" class="break-words text-xs text-red-600 dark:text-red-400">{{ sample.error }}</p>
          <dl class="grid grid-cols-3 gap-2 border-t border-gray-100 pt-3 text-xs dark:border-dark-600"><div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.probeDuration') }}</dt><dd class="mt-1 tabular-nums">{{ seconds(sample.duration_ms) }}</dd></div><div><dt class="text-gray-500 dark:text-gray-400">TTFT</dt><dd class="mt-1 tabular-nums">{{ seconds(sample.ttft_ms) }}</dd></div><div><dt class="text-gray-500 dark:text-gray-400">TPS</dt><dd class="mt-1 tabular-nums">{{ numeric(sample.tps) }}</dd></div></dl>
        </div>
      </article>
    </div>
    <section class="card overflow-hidden" :aria-label="t('admin.modelIdentity.candidateRanking')">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 p-4 dark:border-dark-600"><h3 class="font-semibold">{{ t('admin.modelIdentity.candidateRanking') }}</h3><span class="text-xs text-gray-500 dark:text-gray-400">{{ ranking.length }} {{ t('admin.modelIdentity.candidates') }}</span></div>
      <p v-if="!ranking.length" class="p-6 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.noRanking') }}</p>
      <ol v-else class="divide-y divide-gray-100 dark:divide-dark-600"><li v-for="(candidate, index) in visibleRanking" :key="candidate.model" class="flex min-h-14 items-center gap-3 px-4 py-3"><span class="w-6 shrink-0 text-xs tabular-nums text-gray-400">{{ index + 1 }}</span><div class="min-w-0 flex-1"><p class="break-words text-sm font-medium text-gray-900 dark:text-white">{{ displayModel(candidate.model, candidate.display_name) }}</p><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ candidate.family_name }}</p></div><span class="text-sm tabular-nums">{{ percent(candidate.probability) }}</span></li></ol>
      <button v-if="ranking.length > 8" class="btn btn-ghost min-h-11 w-full rounded-none" type="button" :aria-expanded="showAll" @click="showAll = !showAll">{{ t(showAll ? 'common.collapse' : 'admin.modelIdentity.showAllCandidates') }}</button>
    </section>
    <details v-if="report.tokenizer" class="card p-4"><summary class="min-h-11 cursor-pointer text-sm">{{ t('admin.modelIdentity.tokenizerAuxiliary') }}</summary><pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words text-xs">{{ JSON.stringify(report.tokenizer, null, 2) }}</pre></details>
    <details class="card p-4"><summary class="min-h-11 cursor-pointer text-sm">{{ t('admin.modelIdentity.evidence') }}</summary><dl class="mb-4 grid gap-2 text-xs sm:grid-cols-2"><div>{{ t('admin.modelIdentity.engineVersion') }}: {{ report.engine_commit }}</div><div>{{ t('admin.modelIdentity.baselineVersion') }}: {{ report.benchmark_version }}</div></dl><pre class="max-h-96 overflow-auto whitespace-pre-wrap break-words text-xs">{{ JSON.stringify({ report, requests: run.probes, options: run.options }, null, 2) }}</pre><a class="mt-3 inline-block min-h-11 text-sm text-primary-600 underline dark:text-primary-400" href="https://github.com/Ikaleio/lm-detector/tree/654c8334bb53951aaf04cbd61252d163649e1369" target="_blank" rel="noopener noreferrer">Ikaleio/lm-detector · MIT</a></details>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { identityFingerprintClass, identityRunStatus, identityStatusClass } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
type Candidate = { model: string; display_name: string; family_name: string; probability: number | null; score: number }
type Sample = { id: string; prompt: string; text: string; state: string; status?: string; error?: string; expected_count: number; valid_numbers: number; duration_ms: number | null; ttft_ms: number | null; tps: number | null }
type Analysis = { prediction: string; prediction_name: string; probability: number | null; probability_status: string; decision?: string; results: Candidate[]; used_outputs: number; evidence: { reason: string } }
const props = defineProps<{ run: IdentityRun }>()
const { t } = useI18n()
const report = computed(() => props.run.report || {})
const analysis = computed(() => report.value.analysis as Analysis | undefined)
const samples = computed(() => Array.isArray(report.value.samples) ? report.value.samples as Sample[] : [])
const ranking = computed(() => analysis.value?.results || [])
const showAll = ref(false)
watch(() => props.run.id, () => { showAll.value = false })
const visibleRanking = computed(() => showAll.value ? ranking.value : ranking.value.slice(0, 8))
const isAstra = computed(() => ['gpt-6-astra', 'gpt-6-astra/6.1-sol'].includes(analysis.value?.prediction || ''))
function displayModel(id: string, name?: string) { return id === 'gpt-6-astra' ? 'gpt-6-astra/6.1-sol' : name || id }
const candidateDiffers = computed(() => Boolean(analysis.value?.prediction && props.run.expected_model && identityFingerprintClass(analysis.value.prediction) !== identityFingerprintClass(props.run.expected_model)))
const prediction = computed(() => analysis.value?.prediction ? displayModel(analysis.value.prediction, analysis.value.prediction_name) : analysis.value?.prediction_name || '')
const warnings = computed(() => {
  const a = analysis.value, values: string[] = []
  if (!a) return values
  if (a.used_outputs > 0 && a.used_outputs < 3) values.push(t('admin.modelIdentity.partialWarning', { count: a.used_outputs }))
  if (a.decision === 'not_model') values.push(t('admin.modelIdentity.anomalousWarning'))
  if (a.results.length && a.probability_status !== 'reference_calibrated') values.push(t('admin.modelIdentity.calibrationWarning'))
  if (typeof a.probability === 'number' && a.probability < 0.75) values.push(t('admin.modelIdentity.lowConfidenceWarning'))
  return values
})
function percent(value: unknown) { return typeof value === 'number' && Number.isFinite(value) ? `${(value * 100).toFixed(1)}%` : '—' }
function seconds(value: unknown) { return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? `${(value / 1000).toFixed(2)} s` : '—' }
function numeric(value: unknown) { return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value.toFixed(1) : '—' }
</script>
