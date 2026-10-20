"""Real HTTP management tests against the guarded disposable local gateway."""
from local_guard import guard, sql
guard()
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import urllib.error
import uuid
import http_check as h

h.S['admin_token'] = h.api('POST', '/auth/login', {'email': 'admin@rewards.test', 'password': 'RewardsLocalDemo123'})['access_token']
results = []

def record(name, **facts):
    results.append({'scenario': name, **facts})
    Path('/private/tmp/rewards-local/management-http-results.json').write_text(json.dumps(results, indent=2))
    print('PASS', name, flush=True)

def concurrent(fn, n=12):
    with ThreadPoolExecutor(max_workers=n) as pool:
        return list(pool.map(fn, range(n)))

def denied(fn):
    try:
        fn()
    except urllib.error.HTTPError as err:
        assert err.code in (400, 403, 404, 409), err.code
        return err.code
    raise AssertionError('Expected a rejected operation')

user, token, _ = h.user('management-race')
template = h.admin('GET', '/admin/rewards/templates')[0].copy()
template.update(id=0, version=0, name='Management HTTP fixture', published=True)
template = h.admin('PUT', '/admin/rewards/templates', template)
payload = {'template_id': template['id'], 'count': 3, 'days': 7, 'request_key': str(uuid.uuid4())}
batches = concurrent(lambda _: h.admin('POST', '/admin/rewards/codes', payload)['codes'])
assert all(v == batches[0] for v in batches)
assert len(set(batches[0])) == 3
assert int(sql(f"SELECT COUNT(*) FROM reward_redemptions WHERE template_id={template['id']}")) == 3
denied(lambda: h.admin('POST', '/admin/rewards/codes', {**payload, 'count': 4}))
record('12_code_generation_retries', generated=3)

job = {'user_id': user['id'], 'template_id': template['id'], 'request_key': str(uuid.uuid4()), 'reason': 'Management HTTP retry check'}
cards = concurrent(lambda _: h.admin('POST', '/admin/rewards/grant', job)['card_id'])
assert len(set(cards)) == 1
assert int(sql(f"SELECT COUNT(*) FROM reward_cards WHERE user_id={user['id']} AND source='admin'")) == 1
denied(lambda: h.admin('POST', '/admin/rewards/grant', {**job, 'template_id': 1}))
concurrent(lambda _: h.admin('POST', f'/admin/rewards/cards/{cards[0]}/revoke', {'reason': 'Local test complete'}))
assert sql(f"SELECT COUNT(*) FROM reward_events WHERE card_id={cards[0]} AND kind='revoked'") == '1'
record('12_grant_and_revoke_retries', cards=1, revoke_events=1)

template['published'] = False
h.admin('PUT', '/admin/rewards/templates', template)
assert h.admin('POST', '/admin/rewards/codes', payload)['codes'] == batches[0]
assert h.admin('POST', '/admin/rewards/grant', job)['card_id'] == cards[0]
record('receipts_survive_template_unpublish')

code = batches[0][0]
code_id = int(sql("SELECT id FROM redeem_codes WHERE code='" + code + "'"))
h.admin('PUT', f'/admin/rewards/codes/{code_id}/status', {'status': 'disabled'})
denied(lambda: h.api('POST', '/redeem', {'code': code}, token))
h.admin('PUT', f'/admin/rewards/codes/{code_id}/status', {'status': 'unused'})
h.api('POST', '/redeem', {'code': code}, token)
denied(lambda: h.api('POST', '/redeem', {'code': code}, token))
denied(lambda: h.admin('PUT', f'/admin/rewards/codes/{code_id}/status', {'status': 'unused'}))
denied(lambda: h.admin('POST', '/admin/redeem-codes/batch-update', {'ids': [code_id], 'fields': {'status': 'unused'}}))
assert sql(f"SELECT status FROM redeem_codes WHERE id={code_id}") == 'used'
assert sql(f"SELECT COUNT(*) FROM reward_cards WHERE source_key='code:{code_id}'") == '1'
record('disable_restore_redeem_and_legacy_reopen_rejected', cards=1)

# Competing administrative disable and actual user redemption must have one winner.
template['version'] += 1
template['published'] = True
template = h.admin('PUT', '/admin/rewards/templates', template)
race_codes = h.admin('POST', '/admin/rewards/codes', {'template_id': template['id'], 'count': 8, 'days': 7, 'request_key': str(uuid.uuid4())})['codes']
for index, code in enumerate(race_codes):
    code_id = int(sql("SELECT id FROM redeem_codes WHERE code='" + code + "'"))
    def action(i):
        try:
            if i == 0:
                h.api('POST', '/redeem', {'code': code}, token)
            else:
                h.admin('PUT', f'/admin/rewards/codes/{code_id}/status', {'status': 'disabled'})
            return True
        except urllib.error.HTTPError as err:
            assert err.code in (400, 403, 409), err.code
            return False
    outcomes = concurrent(action, 2)
    assert sum(outcomes) == 1, outcomes
    state = sql(f"SELECT status FROM redeem_codes WHERE id={code_id}")
    count = int(sql(f"SELECT COUNT(*) FROM reward_cards WHERE source_key='code:{code_id}'"))
    assert (state, count) == (('used', 1) if outcomes[0] else ('disabled', 0))
record('8_redemption_disable_races', duplicate_cards=0)
print(json.dumps(results, indent=2))
