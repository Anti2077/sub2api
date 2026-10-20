from http.server import ThreadingHTTPServer,BaseHTTPRequestHandler
import json,uuid,time
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*args):pass
 def do_POST(self):
  body=json.loads(self.rfile.read(int(self.headers.get('Content-Length','0'))));model=body.get('model','gpt-4o-mini');rid='mock_'+uuid.uuid4().hex
  tokens={'input_tokens':1000,'output_tokens':500,'total_tokens':1500,'input_tokens_details':{'cached_tokens':200}}
  out={'id':rid,'object':'response','created_at':int(time.time()),'status':'completed','model':model,'output':[{'id':'msg_'+rid,'type':'message','status':'completed','role':'assistant','content':[{'type':'output_text','text':'本地模拟上游：晶能已点亮。','annotations':[]}]}],'usage':tokens}
  if '/chat/completions' in self.path:
   out={'id':rid,'object':'chat.completion','created':int(time.time()),'model':model,'choices':[{'index':0,'message':{'role':'assistant','content':'本地模拟上游：晶能已点亮。'},'finish_reason':'stop'}],'usage':{'prompt_tokens':1000,'completion_tokens':500,'total_tokens':1500,'prompt_tokens_details':{'cached_tokens':200}}}
  self.send_response(200);self.send_header('x-request-id',rid)
  if body.get('stream'):
   self.send_header('Content-Type','text/event-stream');self.end_headers()
   if '/chat/completions' in self.path:
    chunk={**out,'object':'chat.completion.chunk','choices':[{'index':0,'delta':{'role':'assistant','content':'本地模拟上游：晶能已点亮。'},'finish_reason':None}]}
    self.wfile.write(('data: '+json.dumps(chunk)+'\n\n').encode());self.wfile.flush()
    chunk={**out,'object':'chat.completion.chunk','choices':[{'index':0,'delta':{},'finish_reason':'stop'}]}
    self.wfile.write(('data: '+json.dumps(chunk)+'\n\ndata: [DONE]\n\n').encode());self.wfile.flush();return
   for event,data in [('response.created',{'type':'response.created','response':{**out,'output':[],'status':'in_progress'}}),('response.output_text.delta',{'type':'response.output_text.delta','item_id':'msg_'+rid,'output_index':0,'content_index':0,'delta':'本地模拟上游：晶能已点亮。'}),('response.completed',{'type':'response.completed','response':out})]:
    self.wfile.write(('event: '+event+'\ndata: '+json.dumps(data,ensure_ascii=False)+'\n\n').encode());self.wfile.flush()
   return
  self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(json.dumps(out).encode())
ThreadingHTTPServer(('127.0.0.1',19086),Handler).serve_forever()
