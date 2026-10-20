import json,urllib.request,urllib.error,uuid,pathlib
BASE='http://127.0.0.1:18086/api/v1'
statepath=pathlib.Path('/private/tmp/rewards-local/state.json')
state=json.loads(statepath.read_text()) if statepath.exists() else {}
def save():statepath.write_text(json.dumps(state))
def api(method,path,data=None,token=None):
 h={'Content-Type':'application/json'}
 if token:h['Authorization']='Bearer '+token
 req=urllib.request.Request(BASE+path,data=json.dumps(data).encode() if data is not None else None,headers=h,method=method)
 try:
  r=urllib.request.urlopen(req,timeout=40);v=json.load(r)
 except urllib.error.HTTPError as e:raise RuntimeError(f'{method} {path}: {e.code} {e.read().decode()}')
 if v.get('code')!=0:raise RuntimeError(f'{path}: {v}')
 return v['data']
def admin(method,path,data=None):return api(method,path,data,state['admin_token'])
login=api('POST','/auth/login',{'email':'admin@rewards.test','password':'RewardsLocalDemo123'});state['admin_token']=login['access_token'];save()
compliance=admin('GET','/admin/compliance')
if not compliance.get('acknowledged'):admin('POST','/admin/compliance/accept',{'phrase':compliance['ack_phrase_en'],'language':'en'})
if 'group' not in state:
 state['group']=admin('POST','/admin/groups',{'name':'本机模拟 GPT 农场','platform':'openai','rate_multiplier':0.35,'subscription_type':'standard','model_pricing':[{'platform':'openai','models':['gpt-4o-mini','gpt-6-luna'],'billing_mode':'token','input_price':2,'output_price':10,'cache_read_price':1,'cache_write_price':3}]});save()
gid=state['group']['id']
if 'account' not in state:
 state['account']=admin('POST','/admin/accounts',{'name':'本地协议模拟（不会调用真实上游）','platform':'openai','type':'apikey','credentials':{'api_key':'local-mock-upstream-only','base_url':'http://127.0.0.1:19086'},'concurrency':50,'priority':1,'rate_multiplier':0.2,'group_ids':[gid]});save()
if 'user' not in state:
 state['user']=admin('POST','/admin/users',{'email':'farmer@rewards.test','username':'星光农友','password':'RewardsLocalDemo123','role':'user','balance':100,'concurrency':50,'allowed_groups':[gid]});save()
login=api('POST','/auth/login',{'email':'farmer@rewards.test','password':'RewardsLocalDemo123'});state['user_token']=login['access_token'];save()
if 'key' not in state:state['key']=api('POST','/keys',{'name':'本机农场演示','group_id':gid},state['user_token']);save()
if 'neighbor' not in state:
 state['neighbor']=admin('POST','/admin/users',{'email':'neighbor@rewards.test','username':'月光小园','password':'RewardsLocalDemo123','role':'user','balance':100,'concurrency':5,'allowed_groups':[gid]});save()
login=api('POST','/auth/login',{'email':'neighbor@rewards.test','password':'RewardsLocalDemo123'});state['neighbor_token']=login['access_token'];save()
cfg=admin('GET','/admin/rewards/config');cfg.update(enabled=True,farm_enabled=True);admin('PUT','/admin/rewards/config',cfg)
if 'templates' not in state:
 state['templates']=[]
 for kind,name,tokens in [('model_day','Luna 灵感日卡',0),('group_rate','GPT 分组 0.25× 日卡',0),('model_tokens','Luna 2500 Token 卡',2500)]:
  v=admin('PUT','/admin/rewards/templates',{'name':name,'kind':kind,'model':'gpt-6-luna','group_ids':[gid],'rate':0.25,'tokens':tokens,'hours':720 if tokens else 24,'claim_days':30,'published':True});state['templates'].append(v)
 save()
if 'cards' not in state:
 state['cards']=[]
 for t in state['templates']:
  v=admin('POST','/admin/rewards/grant',{'user_id':state['user']['id'],'template_id':t['id'],'request_key':str(uuid.uuid4()),'reason':'本机验收演示卡'});state['cards'].append(v)
 save()
if 'shop' not in state:
 state['shop']=admin('PUT','/admin/rewards/shop',{'template_id':state['templates'][2]['id'],'price':20,'stock':100,'daily_limit':3,'total_limit':10,'published':True});save()
for who in ['user','neighbor']:
 token=state[who+'_token'];api('POST','/rewards/actions',{'kind':'privacy','public':True,'request_key':str(uuid.uuid4())},token)
 farm=api('GET','/rewards/farm',token=token)
 for p in farm['plots'][:4]:
  if not p['crop'] or p['harvested_at']:api('POST','/rewards/actions',{'kind':'plant','slot':p['slot'],'crop_id':['sprout','cluster','bloom','moon'][p['slot']-1],'request_key':str(uuid.uuid4())},token)
print('Local seed ready: users, group, mock upstream, three templates, cards and farm shop.')
