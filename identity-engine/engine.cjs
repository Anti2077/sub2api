'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { decodeDetector } = require('./dist/shared/detector-codec.js');
const { analyzeSharedOutputs } = require('./dist/shared/shared-detector.js');
const { testApi } = require('./dist/shared/detection.js');
const { generateChallenges } = require('./dist/shared/challenge-browser.js');
const { parseNumbers } = require('./dist/shared/fingerprint-core.js');
const { supportsSharedDetector } = require('./dist/shared/shared-detector.js');
const { probeTokenizer, tokenizerReport } = require('./dist/shared/tokenizer-probe.js');
const { assertTokenizerBank } = require('./dist/shared/tokenizer-bank.js');
const manifest = require('./verify.cjs').verify();
const COMMIT = '654c8334bb53951aaf04cbd61252d163649e1369';
const SOURCE = 'https://github.com/Ikaleio/lm-detector';
const data = name => fs.readFileSync(path.join(__dirname, 'vendor/lm-detector/data', name));
const bank = JSON.parse(data('unified_bank.json'));
const detector = decodeDetector(new Uint8Array(data('shared_detector.bin')));
if (!supportsSharedDetector(bank, detector) || bank.reference_sha256 !== detector.source_reference_sha256) throw new Error('Fingerpoint detector/reference binding mismatch');
const tokenizerBank = JSON.parse(data('tokenizer_bank.json')); assertTokenizerBank(tokenizerBank);
const models = bank.models.map(m => ({ id: m.id, name: m.display_name, family: m.family, family_name: m.family_name }));
function safe(value) {
  if (typeof value === 'string') return value.replace(/sk-[\w-]+/g, '[REDACTED]');
  if (Array.isArray(value)) return value.map(safe);
  if (value && typeof value === 'object') return Object.fromEntries(Object.entries(value).filter(([k]) => !/^(api[_-]?key|authorization|refresh[_-]?token)$/i.test(k)).map(([k, v]) => [k, safe(v)]));
  return value;
}
async function runFingerprint({ expected_model, request_model, options = {}, probeURL, token, signal, emit, transportOverride, challengesOverride }) {
  const challenges = challengesOverride || generateChallenges(3); let latest, lastEmit = 0;
  const timings = new Map();
  const sampleReport = progress => challenges.map((challenge, i) => {
    const state = progress?.challenges?.[i] || { text: '', state: 'pending' };
    const timing = timings.get(i);
    if (timing && !timing.finished && ['done', 'capped', 'rejected', 'stopped'].includes(state.state)) timing.finished = performance.now();
    return { ...challenge, ...state, valid_numbers: parseNumbers(state.text).length, duration_ms: timing?.finished ? timing.finished - timing.started : null,
      ttft_ms: state.throughput?.ttftMs ?? null, tps: state.throughput?.estimated ? null : state.throughput?.tokensPerSecond ?? null };
  });
  const metadata = { engine: 'lm-fingerpoint-detector', engine_commit: COMMIT, source: SOURCE, source_commit: COMMIT, engine_version: 'shared-detector-v2', benchmark_version: bank.built_at,
    artifacts: manifest.files, detector_version: { base_sha256: detector.base_sha256, reference_sha256: detector.source_reference_sha256, calibration_sha256: detector.calibration_sha256 }, expected_model, request_model, options };
  let tokenizer;
  const reportProgress = progress => {
    latest = progress;
    progress.challenges?.forEach((s, i) => { if (s.state === 'requesting' && !timings.has(i)) timings.set(i, { started: performance.now() }); });
    const samples = sampleReport(progress);
    if (performance.now() - lastEmit < 200 && progress.completed !== 3) return;
    lastEmit = performance.now(); emit({ type: 'progress', report: { ...metadata, progress: safe(progress), samples: safe(samples), tokenizer } });
  };
  const format = options.protocol === 'responses' ? 'responses' : options.protocol === 'messages' ? 'anthropic' : 'openai';
  const config = { baseUrl: 'https://model-identity.invalid/v1', apiKey: token, model: request_model, effort: options.reasoning_effort || '', serviceTier: options.service_tier || 'default', format, stream: options.stream ?? true, parallel: options.parallel ?? true, relaxed: options.relaxed ?? true };
  const transport = async (url, config, body, requestSignal) => {
    // Exactly one transport attempt. Gateway handlers own the shared retry
    // budget; the worker never replays a paid request or receives a permanent Key.
    const endpoint = format === 'responses' ? 'responses' : format === 'anthropic' ? 'messages' : 'chat/completions';
    const limit = format === 'responses' ? 'max_output_tokens' : 'max_tokens';
    const payload = { ...body, [limit]: Math.min(Number(body[limit] || 8192), 8192) };
    return fetch(`${probeURL}/${endpoint}`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: JSON.stringify(payload), signal: AbortSignal.any([requestSignal, AbortSignal.timeout(180000)]), redirect: 'error' });
  };
  emit({ type: 'progress', report: { ...metadata, samples: sampleReport() } });
  let outputs;
  try { outputs = await testApi(config, challenges, reportProgress, signal, transportOverride || transport); }
  catch (error) { if (signal?.aborted) throw error; outputs = latest?.outputs || challenges.map(c => ({ text: '', expected_count: c.expected_count })); }
  const analysis = analyzeSharedOutputs(outputs, bank, detector, { allowPartial: true });
  if (options.tokenizer && !signal?.aborted) {
    const startedAt = Date.now();
    const run = await probeTokenizer(config, tokenizerBank, { url: config.baseUrl, transport: transportOverride || transport, signal, concurrency: config.parallel ? 3 : 1,
      onUpdate: run => { tokenizer = tokenizerReport(run, tokenizerBank, { createdAt: startedAt, model: request_model }); emit({ type: 'progress', report: { ...metadata, analysis, samples: sampleReport(latest), tokenizer } }); } });
    tokenizer = tokenizerReport(run, tokenizerBank, { createdAt: startedAt, model: request_model });
  }
  return { ...metadata, detected_model: analysis.prediction || null, ranking: analysis.results, confidence: analysis.probability ?? null, analysis: safe(analysis), samples: safe(sampleReport(latest)), progress: safe(latest), tokenizer: safe(tokenizer) };
}
module.exports = { COMMIT, SOURCE, bank, detector, models, runFingerprint, analyzeSharedOutputs, safe };
