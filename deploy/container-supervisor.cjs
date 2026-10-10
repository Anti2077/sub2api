'use strict';
const { spawn } = require('node:child_process');
const { setTimeout: delay } = require('node:timers/promises');

async function supervise({
  backend = ['/app/sub2api'],
  worker = [process.execPath, '/app/identity-engine/server.cjs'],
  env = process.env,
  startupTimeout = 30000,
  shutdownTimeout = 10000,
} = {}) {
  const callback = `http://127.0.0.1:${env.SERVER_PORT || 8080}`;
  const engine = 'http://127.0.0.1:8081';
  const children = new Set();
  let stopping = false;
  let exitCode = 1;
  let settle;
  const finished = new Promise(resolve => { settle = resolve; });
  let killTimer;
  const stop = (code, reason) => {
    if (stopping) return;
    stopping = true;
    exitCode = code;
    console.error(`[container] ${reason}`);
    for (const child of children) child.kill('SIGTERM');
    if (!children.size) return settle(exitCode);
    killTimer = setTimeout(() => {
      for (const child of children) child.kill('SIGKILL');
    }, shutdownTimeout);
  };
  const start = (name, command, childEnv) => {
    const child = spawn(command[0], command.slice(1), { env: childEnv, stdio: 'inherit' });
    children.add(child);
    child.once('error', () => stop(1, `${name} could not start`));
    child.once('close', (code, signal) => {
      children.delete(child);
      if (!stopping) stop(code || 1, `${name} exited (${signal || code})`);
      if (stopping && !children.size) settle(exitCode);
    });
    return child;
  };
  const onTerm = () => stop(0, 'shutdown requested');
  process.on('SIGTERM', onTerm);
  process.on('SIGINT', onTerm);
  try {
    start('identity worker', worker, {
      PATH: env.PATH, TZ: env.TZ || 'UTC', PORT: '8081', MODEL_IDENTITY_HOST: '127.0.0.1',
      MODEL_IDENTITY_CALLBACK_ORIGIN: env.MODEL_IDENTITY_CALLBACK_URL || callback,
    });
    const deadline = Date.now() + startupTimeout;
    let ready = false;
    while (!stopping && Date.now() < deadline) {
      try {
        const response = await fetch(`${engine}/models`, { signal: AbortSignal.timeout(1000) });
        const catalog = await response.json();
        ready = response.ok && catalog.engine_commit === '5c41136741ca52b5637879cca7bd0cae07404646' && Array.isArray(catalog.models);
        if (ready) break;
      } catch { /* The worker may still be loading its baselines. */ }
      await delay(100);
    }
    if (!stopping) {
      if (!ready) stop(1, 'identity worker startup timed out');
      else start('Sub2API', backend, {
        ...env,
        MODEL_IDENTITY_ENGINE_URL: env.MODEL_IDENTITY_ENGINE_URL || engine,
        MODEL_IDENTITY_CALLBACK_URL: env.MODEL_IDENTITY_CALLBACK_URL || callback,
        MODEL_IDENTITY_REMOTE_API_URL: env.MODEL_IDENTITY_REMOTE_API_URL || 'https://bazaarlink.ai/api/probe/run',
      });
    }
    return await finished;
  } finally {
    clearTimeout(killTimer);
    process.removeListener('SIGTERM', onTerm);
    process.removeListener('SIGINT', onTerm);
  }
}

if (require.main === module) {
  supervise().then(code => { process.exitCode = code; }).catch(() => {
    console.error('[container] supervisor failed');
    process.exitCode = 1;
  });
}
module.exports = { supervise };
