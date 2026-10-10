#!/bin/sh
set -eu

image=${1:-sub2api-identity-single:local}
prefix="sub2api-identity-smoke-$$"
cleanup() {
    docker rm -f "$prefix-app" "$prefix-db" "$prefix-redis" >/dev/null 2>&1 || true
    docker network rm "$prefix" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
docker network create "$prefix" >/dev/null
docker run -d --name "$prefix-db" --network "$prefix" --network-alias postgres \
    --tmpfs /var/lib/postgresql -e POSTGRES_PASSWORD=local-smoke-password \
    -e POSTGRES_USER=sub2api -e POSTGRES_DB=sub2api postgres:18-alpine >/dev/null
docker run -d --name "$prefix-redis" --network "$prefix" --network-alias redis \
    redis:8-alpine >/dev/null
for attempt in $(seq 1 60); do
    if docker exec "$prefix-db" pg_isready -U sub2api >/dev/null 2>&1; then break; fi
    sleep 1
done
docker run -d --name "$prefix-app" --network "$prefix" \
    --tmpfs /app/data:uid=1000,gid=1000 \
    -e AUTO_SETUP=true -e DATABASE_HOST=postgres -e DATABASE_SSLMODE=disable \
    -e DATABASE_USER=sub2api -e DATABASE_PASSWORD=local-smoke-password \
    -e DATABASE_DBNAME=sub2api -e REDIS_HOST=redis \
    -e ADMIN_EMAIL=admin@example.test -e ADMIN_PASSWORD=LocalSmokePassword123 \
    -e JWT_SECRET=local-smoke-only-secret-0123456789abcdef \
    "$image" >/dev/null
ready=false
for attempt in $(seq 1 90); do
    if docker exec "$prefix-app" wget -q -O /dev/null http://127.0.0.1:8080/health 2>/dev/null; then ready=true; break; fi
    sleep 1
done
if [ "$ready" != true ]; then
    docker logs "$prefix-app"
    exit 1
fi
docker exec -i "$prefix-app" node <<'JS'
const assert = require('node:assert/strict');
const fs = require('node:fs');
(async () => {
  const worker = await fetch('http://127.0.0.1:8081/models').then(r => r.json());
  assert.equal(worker.engine_commit, '5c41136741ca52b5637879cca7bd0cae07404646');
  assert.ok(worker.models.length > 0);
  // Embedded frontend middleware must not turn protected callbacks into HTML.
  for (const path of ['/internal/model-identity/0/remote/v1/chat/completions', '/internal/model-identity/0/probe']) {
    const callback = await fetch(`http://127.0.0.1:8080${path}`, {
      method: 'POST', headers: {'Content-Type': 'application/json'}, body: '{}',
    });
    assert.equal(callback.status, 401, `unauthenticated callback must reach its guard: ${path}`);
    assert.ok(!(callback.headers.get('Content-Type') || '').includes('text/html'));
  }
  const login = await fetch('http://127.0.0.1:8080/api/v1/auth/login', {
    method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({email: 'admin@example.test', password: 'LocalSmokePassword123'}),
  }).then(r => r.json());
  assert.ok(login.data.access_token);
  const compliance = await fetch('http://127.0.0.1:8080/api/v1/admin/compliance/accept', {
    method: 'POST', headers: {'Content-Type': 'application/json', Authorization: `Bearer ${login.data.access_token}`},
    body: JSON.stringify({language: 'en', phrase: 'I have read, understood, and agree to the Sub2API Deployment and Operation Compliance Commitment'}),
  });
  if (compliance.status !== 200) throw new Error(`compliance acknowledgement returned ${compliance.status}: ${await compliance.text()}`);
  const response = await fetch('http://127.0.0.1:8080/api/v1/admin/model-identity/models', {
    headers: {Authorization: `Bearer ${login.data.access_token}`},
  });
  if (response.status !== 200) throw new Error(`admin model catalog returned ${response.status}: ${await response.text()}`);
  const catalog = await response.json();
  assert.ok(Array.isArray(catalog.models) && catalog.models.length > 0);
  for (const pid of fs.readdirSync('/proc').filter(p => /^\d+$/.test(p))) {
    try {
      const cmd = fs.readFileSync(`/proc/${pid}/cmdline`, 'utf8');
      if (cmd.includes('/app/identity-engine/server.cjs') || cmd === '/app/sub2api\0') {
        assert.match(fs.readFileSync(`/proc/${pid}/status`, 'utf8'), /Uid:\s+1000\s+1000/);
      }
    } catch (e) { if (e.code !== 'ENOENT' && e.code !== 'ESRCH') throw e; }
  }
  assert.ok(fs.existsSync('/app/identity-engine/vendor/bazaarlink/LICENSE'));
  console.log(`single image: admin catalog has ${catalog.models.length} models; worker and backend run as UID 1000`);
})().catch(e => { console.error(`single image API or runtime assertion failed: ${e.message}`); process.exitCode = 1; });
JS
docker exec "$prefix-app" /app/docker-entrypoint.sh --version >/dev/null
docker stop -t 15 "$prefix-app" >/dev/null
test "$(docker inspect --format '{{.State.ExitCode}}' "$prefix-app")" = 0
printf 'single image startup, admin catalog, CLI and graceful stop passed\n'
