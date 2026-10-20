"""Real HTTP pre-provider idempotency checks, using only disposable loopback fixtures."""
from local_guard import guard
guard()
import http_check as h
import concurrent.futures, json, pathlib, time, uuid, urllib.request, urllib.error
ROOT=pathlib.Path('/private/tmp/rewards-local')
def calls():
 p=ROOT/'mock-calls.jsonl'
 return len(p.read_text().splitlines()) if p.exists() else 0
def send(key,idem,body):
 req=urllib.request.Request(h.BASE+'/v1/responses',data=json.dumps(body).encode(),headers={'Content-Type':'application/json','Authorization':'Bearer '+key['key'],'Idempotency-Key':idem})
 try:
  with urllib.request.urlopen(req,timeout=40) as r:return r.status,dict(r.headers),r.read()
 except urllib.error.HTTPError as e:return e.code,dict(e.headers),e.read()
def verify():
 (ROOT/'mock-control.json').write_text('{}')
 u,t,k=h.user('replay');card=h.grant(u,t,'model_tokens',3000)
 body={'model':'gpt-6-luna','input':'Local idempotency verification','max_output_tokens':500}
 idem=str(uuid.uuid4());before=calls();a=send(k,idem,body);b=send(k,idem,body)
 assert a[0]==b[0]==200,(a[0],b[0],b[2][:200])
 assert a[2]==b[2] and b[1].get('Idempotency-Replayed')=='true',b
 assert calls()==before+1,(calls(),before)
 assert send(k,idem,{**body,'input':'changed'})[0]==409
 assert calls()==before+1
 (ROOT/'mock-control.json').write_text('{"delay":1}')
 idem2=str(uuid.uuid4());before=calls()
 with concurrent.futures.ThreadPoolExecutor(max_workers=20) as p: rs=list(p.map(lambda _:send(k,idem2,body),range(20)))
 assert all(r[0] in [200,409] for r in rs),[(r[0],r[2][:100]) for r in rs]
 assert any(r[0]==200 for r in rs)
 assert calls()==before+1,(calls(),before)
 deadline=time.time()+10
 while time.time()<deadline:
  n=h.sql(f"SELECT count(*) FROM reward_usage WHERE api_key_id={k['id']}")
  if n=='2':break
  time.sleep(.1)
 assert n=='2',n
 assert h.sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}')=='0'
 assert h.sql(f'SELECT balance FROM users WHERE id={u["id"]}')=='100.00000000'
 state={'key':k,'idempotency_key':idem,'body':body,'response':a[2].decode(),'request_id':a[1].get('X-Client-Request-Id',a[1].get('X-Client-Request-ID'))}
 (ROOT/'replay-state.json').write_text(json.dumps(state))
 result={'identical_replay':'pass','changed_payload':'409','20_concurrent_one_provider_call':'pass','usage_receipts':2,'covered_tokens':3000,'balance':'100.00000000','restart':'not_yet_verified'}
 (ROOT/'replay-results.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
if __name__=='__main__':
 try:verify()
 finally:(ROOT/'mock-control.json').write_text('{}')
