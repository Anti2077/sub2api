"""Verify issued card snapshots through real local HTTP redemption and bag APIs.

Run against the isolated rewards-local fixture after bootstrap. Creates only
@rewards.test users, templates, and codes after local_guard verifies identity.
The gateway billing configuration and original templates are not changed.
"""
import json, uuid, urllib.request, urllib.error
from pathlib import Path
from local_guard import guard,sql
guard()
BASE='http://127.0.0.1:18086/api/v1'
def api(method,path,data=None,token=None):
 headers={'Content-Type':'application/json'}
 if token: headers['Authorization']='Bearer '+token
 req=urllib.request.Request(BASE+path,data=json.dumps(data).encode() if data is not None else None,headers=headers,method=method)
 with urllib.request.urlopen(req,timeout=40) as r: v=json.load(r)
 assert v['code']==0,v
 return v['data']
admin_token=api('POST','/auth/login',{'email':'admin@rewards.test','password':'RewardsLocalDemo123'})['access_token']
def admin(method,path,data=None):return api(method,path,data,admin_token)
group=next(g for g in admin('GET','/admin/groups/all') if g['name']=='本机模拟 GPT 农场')
email='receipt-'+uuid.uuid4().hex[:8]+'@rewards.test'
u=admin('POST','/admin/users',{'email':email,'username':'回执验收-'+uuid.uuid4().hex[:8],'password':'RewardsLocalDemo123','balance':100,'concurrency':5,'allowed_groups':[group['id']]})
token=api('POST','/auth/login',{'email':email,'password':'RewardsLocalDemo123'})['access_token']
results=[]; demo=[]
for kind,benefit in [('model_tokens',{'tokens':1250000}),('group_rate',{'rate':.25}),('model_day',{})]:
 t=admin('PUT','/admin/rewards/templates',{'name':{'model_tokens':'灵感补给 · 125万 Token 卡','group_rate':'轻松畅聊 · 0.25 倍率日卡','model_day':'Luna 灵感日 · 模型免费日卡'}[kind],'kind':kind,'model':'gpt-6-luna','group_ids':[group['id']],'rate':0,'tokens':0,'hours':24,'claim_days':30,'published':True,**benefit})
 codes=admin('POST','/admin/rewards/codes',{'template_id':t['id'],'count':4,'days':7,'request_key':str(uuid.uuid4())})['codes']
 admin('PUT','/admin/rewards/templates',{**t,'name':'Later edited template','tokens':999999 if kind=='model_tokens' else 0})
 before=api('GET','/auth/me',token=token)
 response=api('POST','/redeem',{'code':codes[0]},token)
 receipt=response['card']; assert response['type']=='card' and response['value']==0
 assert receipt['template']==t,(receipt['template'],t)
 assert receipt['user_id']==u['id'] and receipt['status']=='unused' and receipt['activated_at'] is None and receipt['expires_at'] is None
 bag=api('GET','/rewards/cards',token=token)
 assert next(c for c in bag if c['id']==receipt['id'])==receipt
 after=api('GET','/auth/me',token=token)
 assert after['balance']==before['balance'] and after['concurrency']==before['concurrency']
 try:api('POST','/redeem',{'code':codes[0]},token)
 except urllib.error.HTTPError as e: assert e.code in (400,409)
 else:raise AssertionError('Duplicate code succeeded')
 assert sql(f"SELECT count(*) FROM reward_cards WHERE id={receipt['id']} AND source='code'")=='1'
 results.append({'kind':kind,'snapshot_matches':True,'bag_matches':True,'balance_unchanged':True,'inactive':True,'duplicate_rejected':True,'card_id':receipt['id']})
 demo.append({'kind':kind,'codes':codes[1:],'template':t})
Path('/private/tmp/rewards-local/receipt-http-results.json').write_text(json.dumps(results,ensure_ascii=False,indent=2))
Path('/private/tmp/rewards-local/receipt-demo-codes.json').write_text(json.dumps(demo,ensure_ascii=False,indent=2))
print(json.dumps(results,ensure_ascii=False,indent=2))
