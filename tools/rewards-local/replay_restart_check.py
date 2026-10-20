from local_guard import guard
guard()
import json,pathlib,urllib.request,urllib.error,subprocess
import replay_check as r
ROOT=pathlib.Path('/private/tmp/rewards-local'); s=json.loads((ROOT/'replay-state.json').read_text()); before=r.calls()
status,headers,body=r.send(s['key'],s['idempotency_key'],s['body'])
assert status==200 and body.decode()==s['response'],(status,body[:200])
assert headers.get('Idempotency-Replayed')=='true'
assert r.calls()==before,(before,r.calls())
print(json.dumps({'restart_replay':'pass','status':status,'provider_calls_delta':0,'idempotency_replayed':True}))
