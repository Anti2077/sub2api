'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { spawn } = require('node:child_process');
const { once } = require('node:events');
const path = require('node:path');

const supervisor = path.resolve(__dirname, '../container-supervisor.cjs');
const workerScript = `
  const http = require('node:http');
  const server = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({engine_commit:'5c41136741ca52b5637879cca7bd0cae07404646', models:[]}));
  });
  server.listen(Number(process.env.PORT), process.env.MODEL_IDENTITY_HOST, () => console.log('worker-ready ' + JSON.stringify({ secret: process.env.JWT_SECRET || null, host: process.env.MODEL_IDENTITY_HOST })));
  process.on('SIGTERM', () => { console.log('worker-stopped'); server.close(); });
`;
const backendScript = `
  console.log('backend-ready ' + JSON.stringify({engine:process.env.MODEL_IDENTITY_ENGINE_URL, callback:process.env.MODEL_IDENTITY_CALLBACK_URL}));
  const timer = setInterval(() => {}, 1000);
  process.on('SIGTERM', () => { console.log('backend-stopped'); clearInterval(timer); });
`;

function launch(t, { worker = workerScript, backend = backendScript, startupTimeout = 3000 } = {}) {
  const options = {
    worker: [process.execPath, '-e', worker],
    backend: [process.execPath, '-e', backend],
    env: { ...process.env, JWT_SECRET: 'test-only-secret', SERVER_PORT: '8180', MODEL_IDENTITY_ENGINE_URL: '', MODEL_IDENTITY_CALLBACK_URL: '' },
    startupTimeout, shutdownTimeout: 1000,
  };
  const child = spawn(process.execPath, ['-e', `require(${JSON.stringify(supervisor)}).supervise(${JSON.stringify(options)}).then(code => process.exitCode = code)`]);
  let output = '';
  child.stdout.on('data', chunk => { output += chunk; });
  child.stderr.on('data', chunk => { output += chunk; });
  const closed = once(child, 'close');
  t.after(async () => {
    if (child.exitCode === null && !child.signalCode) child.kill('SIGTERM');
    await closed;
  });
  async function waitFor(text) {
    const deadline = Date.now() + 5000;
    while (!output.includes(text) && Date.now() < deadline) {
      if (child.exitCode !== null) break;
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    assert.ok(output.includes(text), `missing ${text}: ${output}`);
  }
  return { child, closed, waitFor, output: () => output };
}

test('starts the worker first, supplies loopback defaults, and stops both on SIGTERM', async t => {
  const run = launch(t);
  await run.waitFor('backend-ready');
  assert.ok(run.output().indexOf('worker-ready') < run.output().indexOf('backend-ready'));
  assert.match(run.output(), /"secret":null,"host":"127\.0\.0\.1"/);
  assert.match(run.output(), /"engine":"http:\/\/127\.0\.0\.1:8081"/);
  assert.match(run.output(), /"callback":"http:\/\/127\.0\.0\.1:8180"/);
  run.child.kill('SIGTERM');
  assert.equal((await run.closed)[0], 0);
  assert.match(run.output(), /worker-stopped/);
  assert.match(run.output(), /backend-stopped/);
});

test('backend failure shuts down the worker and preserves the failure code', async t => {
  const run = launch(t, { backend: "process.exit(7)" });
  assert.equal((await run.closed)[0], 7);
  assert.match(run.output(), /worker-stopped/);
});

test('worker failure shuts down the running backend', async t => {
  const run = launch(t, { worker: workerScript + "setTimeout(() => process.exit(9), 1000);" });
  await run.waitFor('backend-ready');
  assert.equal((await run.closed)[0], 9);
  assert.match(run.output(), /backend-stopped/);
});

test('startup timeout stops the worker without starting the backend', async t => {
  const run = launch(t, {
    worker: "setInterval(() => {}, 1000); process.on('SIGTERM', () => process.exit(0));",
    startupTimeout: 200,
  });
  assert.equal((await run.closed)[0], 1);
  assert.match(run.output(), /startup timed out/);
  assert.doesNotMatch(run.output(), /backend-ready/);
});

test('forces an unresponsive backend to exit after the shutdown grace period', async t => {
  const run = launch(t, { backend: backendScript.replace("clearInterval(timer)", "void timer") });
  await run.waitFor('backend-ready');
  run.child.kill('SIGTERM');
  assert.equal((await run.closed)[0], 0);
  assert.match(run.output(), /backend-stopped/);
});
