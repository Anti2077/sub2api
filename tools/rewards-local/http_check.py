import json,urllib.request,urllib.error,uuid,subprocess,time,concurrent.futures,decimal,pathlib
S=json.load(open('/private/tmp/rewards-local/state.json'));BASE='http://127.0.0.1:18086';results=[]
def api(method,path,data=None,token=None):
 h={'Content-Type':'application/json'}
 if token:h['Authorization']='Bearer '+token
 req=urllib.request.Request(BASE+'/api/v1'+path,data=json.dumps(data).encode() if data is not None else None,headers=h,method=method)
 with urllib.request.urlopen(req,timeout=40) as r:v=json.load(r)
 assert v['code']==0,v
 return v['data']
def admin(method,path,data=None):return api(method,path,data,S['admin_token'])
def sql(q):
 cmd=['docker','compose','-f','/private/tmp/rewards-local/compose.yaml','exec','-T','postgres','psql','-U','rewards_test','-d','rewards_test','-At','-c',q]
 return subprocess.check_output(cmd,text=True).strip()
def user(name,balance=100):
 u=admin('POST','/admin/users',{'email':name+'-'+uuid.uuid4().hex[:6]+'@rewards.test','username':name+uuid.uuid4().hex[:5],'password':'RewardsLocalDemo123','balance':balance,'concurrency':50,'allowed_groups':[S['group']['id']]})
 token=api('POST','/auth/login',{'email':u['email'],'password':'RewardsLocalDemo123'})['access_token'];k=api('POST','/keys',{'name':'HTTP billing test','group_id':S['group']['id']},token);return u,token,k

def grant(u,token,kind,tokens=0):
 tid=next(t['id'] for t in S['templates'] if t['kind']==kind)
 if tokens:
  t=next(t.copy() for t in S['templates'] if t['kind']==kind);t.update(id=0,version=0,tokens=tokens);tid=admin('PUT','/admin/rewards/templates',t)['id']
 card=admin('POST','/admin/rewards/grant',{'user_id':u['id'],'template_id':tid,'request_key':str(uuid.uuid4()),'reason':'真实 HTTP 专项测试'})['card_id']
 api('POST',f'/rewards/cards/{card}/activate',{},token);return card

def request(key,model='gpt-6-luna',stream=False,rid=None):
 rid=rid or str(uuid.uuid4());req=urllib.request.Request(BASE+'/v1/responses',data=json.dumps({'model':model,'input':'本机精确用量对账','max_output_tokens':500,'stream':stream}).encode(),headers={'Content-Type':'application/json','Authorization':'Bearer '+key['key'],'X-Client-Request-ID':rid})
 try:
  with urllib.request.urlopen(req,timeout=40) as r:body=r.read();status=r.status;rid=r.headers['X-Client-Request-ID']
 except urllib.error.HTTPError as e:return e.code,None
 deadline=time.time()+10
 while time.time()<deadline:
  raw=sql(f"SELECT row_to_json(x) FROM (SELECT input_tokens,output_tokens,cache_read_tokens,total_cost,actual_cost FROM usage_logs WHERE api_key_id={key['id']} AND request_id='client:{rid}') x")
  if raw:return status,json.loads(raw)
  time.sleep(.05)
 raise AssertionError('Missing usage '+rid+' '+body.decode()[:200])
def check(name,row,expected):
 assert row is not None,name
 assert decimal.Decimal(str(row['actual_cost']))==decimal.Decimal(expected),(name,row,expected)
 assert (row['input_tokens'],row['output_tokens'],row['cache_read_tokens'])==(800,500,200),(name,row)
 assert decimal.Decimal(str(row['total_cost']))==decimal.Decimal('.0068'),row
 results.append({'scenario':name,'original':'.00238000','paid':expected,'discount':str(decimal.Decimal('.00238')-decimal.Decimal(expected)),'tokens':1500,'upstream_total':'.0068'})
def main():
 u,token,key=user('无卡对照');status,row=request(key);assert status==200;check('no_card',row,'.00238000')
 u,token,key=user('免费日卡零余额',0);card=grant(u,token,'model_day');status,row=request(key);assert status==200,(status,row);check('free_day_zero_balance',row,'0');status,_=request(key,'gpt-4o-mini');assert status==403,status
 u,token,key=user('倍率卡');grant(u,token,'group_rate');status,row=request(key);assert status==200;check('rate_day',row,'.00170000')
 u,token,key=user('Token部分覆盖');card=grant(u,token,'model_tokens',2500);status,row=request(key);check('tokens_full',row,'0');status,row=request(key);check('tokens_partial',row,'.00079333');status,row=request(key);check('tokens_exhausted',row,'.00238000');assert sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}')=='0'
 u,token,key=user('流式Token卡');grant(u,token,'model_tokens',1500);status,row=request(key,stream=True);assert status==200;check('stream_tokens_exact',row,'0')
 u,token,key=user('并发多Key');card=grant(u,token,'model_tokens',3000);key2=api('POST','/keys',{'name':'第二Key','group_id':S['group']['id']},token)
 with concurrent.futures.ThreadPoolExecutor(max_workers=12) as pool:rs=list(pool.map(lambda i:request(key if i%2 else key2),range(12)))
 assert all(code==200 for code,row in rs),rs
 paid=sum(decimal.Decimal(str(row['actual_cost'])) for code,row in rs);assert paid==decimal.Decimal('.0238'),paid
 assert sql(f'SELECT remaining_tokens FROM reward_cards WHERE id={card}')=='0'
 assert decimal.Decimal(sql(f'SELECT balance FROM users WHERE id={u["id"]}'))==decimal.Decimal('99.9762')
 results.append({'scenario':'12_concurrent_two_keys','original':'.02856000','paid':str(paid),'discount':'.00476000','covered_tokens':3000})
 pathlib.Path('/private/tmp/rewards-local/http-results.json').write_text(json.dumps(results,ensure_ascii=False,indent=2));print(json.dumps(results,ensure_ascii=False,indent=2))

if __name__ == "__main__":
 from local_guard import guard
 guard()
 main()
