'use strict';
const http = require('node:http');
const { COMMIT, SOURCE, models, runFingerprint } = require('./engine.cjs');
let active = 0;
const send = (res, status, body) => { res.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' }); res.end(JSON.stringify(body)); };
const callbackOrigin = process.env.MODEL_IDENTITY_CALLBACK_ORIGIN || 'http://127.0.0.1:8080';
const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/models') return send(res, 200, { engine: 'lm-fingerpoint-detector', engine_commit: COMMIT, source: SOURCE, models });
  if (req.method !== 'POST' || req.url !== '/run') return send(res, 404, { error: 'not found' });
  if (active >= 2) return send(res, 503, { error: 'worker capacity reached' });
  let body = '';
  try {
    for await (const chunk of req) { body += chunk; if (body.length > 16384) throw new Error('request too large'); }
    const options = JSON.parse(body), url = new URL(options.probe_url);
    const origin = new URL(callbackOrigin);
    if (!['127.0.0.1', '[::1]'].includes(origin.hostname) || origin.protocol !== 'http:' || url.origin !== origin.origin || url.pathname !== `/internal/model-identity/${options.run_id}/v1` || url.search || url.hash || url.username || url.password || !/^[a-f0-9]{64}$/.test(options.token) || !/^\d+$/.test(String(options.run_id))) return send(res, 400, { error: 'invalid capability or callback' });
    const capabilityURL = new URL(url); capabilityURL.pathname = `/internal/model-identity/${options.run_id}/capability`;
    const capability = await fetch(capabilityURL, { headers: { Authorization: `Bearer ${options.token}` }, signal: AbortSignal.timeout(10000), redirect: 'error' });
    if (!capability.ok) return send(res, 401, { error: 'invalid or expired capability' });
    const binding = await capability.json();
    const controller = new AbortController(), timeout = setTimeout(() => controller.abort(), 20 * 60 * 1000);
    res.on('close', () => { if (!res.writableEnded) controller.abort(); });
    if (active >= 2) { clearTimeout(timeout); return send(res, 503, { error: 'worker capacity reached' }); }
    active++;
    res.writeHead(200, { 'Content-Type': 'application/x-ndjson', 'Cache-Control': 'no-store', Connection: 'keep-alive' });
    const emit = event => { if (!res.writableEnded) res.write(JSON.stringify(event) + '\n'); };
    try { emit({ type: 'started', engine_commit: COMMIT }); const report = await runFingerprint({ ...binding,  probeURL: url.href, token: options.token, signal: controller.signal, emit }); emit({ type: 'complete', report }); }
    catch (error) { emit({ type: 'error', error: error?.message?.replaceAll(options.token, '[REDACTED]') || 'identity execution interrupted', code: error?.code || 'service_error' }); }
    finally { clearTimeout(timeout); active--; if (!res.writableEnded) res.end(); }
  } catch { if (!res.writableEnded) send(res, 400, { error: 'invalid request' }); }
});
server.requestTimeout = 20 * 60 * 1000;
server.listen(Number(process.env.PORT || 8081), '127.0.0.1');
