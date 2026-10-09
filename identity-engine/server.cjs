'use strict';
const http = require('node:http');
const { COMMIT, models, runIdentity } = require('./identity.cjs');
const callbackOrigin = process.env.MODEL_IDENTITY_CALLBACK_ORIGIN;
if (!callbackOrigin) throw new Error('MODEL_IDENTITY_CALLBACK_ORIGIN is required');
let active = 0;
const server = http.createServer(async (req, res) => {
  const send = (status, body) => { res.writeHead(status, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(body)); };
  if (req.method === 'GET' && req.url === '/models') return send(200, { engine_commit: COMMIT, models });
  if (req.method !== 'POST' || req.url !== '/run') return send(404, { error: 'not found' });
  if (active >= 2) return send(503, { error: 'worker capacity reached' });
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 20 * 60 * 1000);
  res.on('close', () => { if (!res.writableEnded) controller.abort(); });
  try {
    let body = '';
    for await (const chunk of req) { body += chunk; if (body.length > 16384) throw new Error('request too large'); }
    const options = JSON.parse(body);
    const url = new URL(options.probe_url);
    if (url.origin !== new URL(callbackOrigin).origin || url.pathname !== `/internal/model-identity/${options.run_id}/probe` || url.search || url.username || url.password ||
        !/^[a-f0-9]{64}$/.test(options.token)) return send(400, { error: 'invalid capability or callback' });
    const capabilityURL = new URL(url);capabilityURL.pathname = `/internal/model-identity/${options.run_id}/capability`;
    const capability = await fetch(capabilityURL, { headers: { Authorization: `Bearer ${options.token}` }, signal: AbortSignal.timeout(10000), redirect: 'error' });
    if (!capability.ok) return send(401, { error: 'invalid or expired capability' });
    const binding = await capability.json();
    options.expected_model = binding.expected_model;options.model = binding.request_model;
    if (active >= 2) return send(503, { error: 'worker capacity reached' });
    active++;
    try {
      const report = await runIdentity({ expected_model: binding.expected_model, signal: controller.signal }, async probe => {
        const response = await fetch(url, { method: 'POST', headers: { Authorization: `Bearer ${options.token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(probe), signal: AbortSignal.any([controller.signal, AbortSignal.timeout(180000)]), redirect: 'error' });
        if (!response.ok) throw new Error(`probe request failed (${response.status})`);
        const text = await response.text();
        if (text.length > 2 * 1024 * 1024) throw new Error('probe response too large');
        return JSON.parse(text);
      });
      send(200, report);
    } finally { active--; }
  } catch { if (!res.writableEnded) send(500, { error: 'identity execution interrupted' }); }
  finally { clearTimeout(timeout); }
});
server.requestTimeout = 20 * 60 * 1000;
server.listen(Number(process.env.PORT || 8081), '0.0.0.0');
