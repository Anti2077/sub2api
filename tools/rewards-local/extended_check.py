"""Disposable HTTP tests; verify the actual backend's cwd/config/connections first."""
from local_guard import guard
guard()
import http_check as h
import concurrent.futures
import json
import pathlib
import time
import uuid
import urllib.request
import urllib.error

ROOT = pathlib.Path('/private/tmp/rewards-local')
results = []


def control(**cfg):
    (ROOT / 'mock-control.json').write_text(json.dumps(cfg))
    (ROOT / 'mock-seen.json').unlink(missing_ok=True)


def record(name, **details):
    results.append({'scenario': name, **details})
    (ROOT / 'extended-results.json').write_text(json.dumps(results, indent=2))
    print('PASS', name, flush=True)


def raw(key, model='gpt-6-luna', protocol='responses', stream=False, rid=None):
    rid = rid or str(uuid.uuid4())
    path = '/v1/' + protocol
    body = {'model': model, 'stream': stream, 'max_tokens': 500}
    if protocol == 'responses':
        body = {'model': model, 'stream': stream, 'input': 'Local billing verification', 'max_output_tokens': 500}
    elif protocol in ['messages', 'chat/completions']:
        body['messages'] = [{'role': 'user', 'content': 'Local billing verification'}]
    else:
        path = f'/v1beta/models/{model}:' + ('streamGenerateContent?alt=sse' if stream else 'generateContent')
        body = {'contents': [{'role': 'user', 'parts': [{'text': 'Local billing verification'}]}]}
    req = urllib.request.Request(h.BASE + path, data=json.dumps(body).encode(), headers={'Content-Type': 'application/json', 'Authorization': 'Bearer ' + key['key'], 'anthropic-version': '2023-06-01', 'X-Client-Request-ID': rid})
    try:
        with urllib.request.urlopen(req, timeout=40) as r:
            return r.status, r.headers.get('X-Client-Request-ID', rid), r.read()
    except urllib.error.HTTPError as e:
        return e.code, rid, e.read()


def usage(key, rid, wait=True):
    deadline = time.time() + (10 if wait else 0)
    while True:
        v = h.sql(f"SELECT row_to_json(x) FROM (SELECT input_tokens,output_tokens,cache_read_tokens,total_cost,actual_cost FROM usage_logs WHERE api_key_id={key['id']} AND request_id='client:{rid}') x")
        if v:
            return json.loads(v)
        if time.time() >= deadline:
            return None
        time.sleep(.1)


def main():
    control()
    for platform, model, protocol in [('openai', 'gpt-6-luna', 'chat/completions'), ('anthropic', 'claude-sonnet-4-5', 'messages'), ('gemini', 'gemini-2.5-flash', 'gemini')]:
        u, tok, key = h.user('protocol')
        group = h.S['group']
        if platform != 'openai':
            group = h.admin('POST', '/admin/groups', {'name': 'HTTP-' + uuid.uuid4().hex[:8], 'platform': platform, 'rate_multiplier': .35, 'subscription_type': 'standard', 'model_pricing': [{'platform': platform, 'models': [model], 'billing_mode': 'token', 'input_price': 2, 'output_price': 10, 'cache_read_price': 1, 'cache_write_price': 3}]})
            h.admin('POST', '/admin/accounts', {'name': 'Loopback-' + uuid.uuid4().hex[:8], 'platform': platform, 'type': 'apikey', 'credentials': {'api_key': 'local-mock-only', 'base_url': 'http://127.0.0.1:19086'}, 'concurrency': 50, 'priority': 1, 'rate_multiplier': .2, 'group_ids': [group['id']]})
            h.admin('PUT', f'/admin/users/{u["id"]}', {'allowed_groups': [group['id']]})
            key = h.api('POST', '/keys', {'name': 'Protocol test', 'group_id': group['id']}, tok)
        t = h.admin('PUT', '/admin/rewards/templates', {'name': 'Protocol tokens', 'kind': 'model_tokens', 'model': model, 'group_ids': [group['id']], 'rate': .25, 'tokens': 3000, 'hours': 720, 'claim_days': 30, 'published': True})
        card = h.admin('POST', '/admin/rewards/grant', {'user_id': u['id'], 'template_id': t['id'], 'request_key': str(uuid.uuid4()), 'reason': 'Local protocol acceptance'})['card_id']
        h.api('POST', f'/rewards/cards/{card}/activate', {}, tok)
        for stream in [False, True]:
            status, rid, payload = raw(key, model, protocol, stream)
            assert status == 200, (platform, status, payload[:500])
            r = usage(key, rid)
            assert r and (r['input_tokens'], r['output_tokens'], r['cache_read_tokens']) == (800, 500, 200), (platform, r)
            assert r['actual_cost'] == 0, (platform, r)
        assert h.sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}') == '0'
        record('protocol_' + platform, requests=2, normalized_tokens=3000)
    for transition in ['revoke', 'expire']:
        u, tok, key = h.user('snapshot')
        card = h.grant(u, tok, 'model_day')
        control(delay=2)
        with concurrent.futures.ThreadPoolExecutor() as pool:
            f = pool.submit(raw, key)
            end = time.time() + 10
            while not (ROOT / 'mock-seen.json').exists() and time.time() < end:
                time.sleep(.05)
            assert (ROOT / 'mock-seen.json').exists()
            if transition == 'revoke':
                h.admin('POST', f'/admin/rewards/cards/{card}/revoke', {'reason': 'Local snapshot acceptance'})
            else:
                h.sql(f"UPDATE reward_cards SET expires_at=NOW()-INTERVAL '1 second' WHERE id={card}")
            status, rid, payload = f.result()
        assert status == 200, payload[:300]
        assert usage(key, rid)['actual_cost'] == 0
        control()
        status, rid, _ = raw(key)
        assert status == 200 and usage(key, rid)['actual_cost'] == .00238
        record('inflight_' + transition, inflight_paid=0, next_paid='.00238000')
    u, tok, key = h.user('failed')
    card = h.grant(u, tok, 'model_tokens', 3000)
    control(error=400)
    status, rid, _ = raw(key)
    assert status >= 400
    time.sleep(.5)
    assert usage(key, rid, False) is None
    assert h.sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}') == '3000'
    record('upstream_failure_without_usage')
    control(cutoff=True)
    status, rid, _ = raw(key, stream=True)
    time.sleep(.5)
    assert h.sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}') == '3000'
    assert h.sql(f'SELECT balance::text FROM users WHERE id={u["id"]}') == '100.00000000'
    record('stream_cutoff_without_usage', status=status)


if __name__ == '__main__':
    try:
        main()
    finally:
        control()
