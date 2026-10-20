"""Loopback protocol fixture; never calls providers. Control file is test-only."""
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from pathlib import Path
import json
import os
import time
import uuid
import threading
CALL_LOCK = threading.Lock()

ROOT = Path(os.environ.get('REWARDS_LOCAL_DIR', '/private/tmp/rewards-local'))


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_POST(self):
        with CALL_LOCK:
            with (ROOT / 'mock-calls.jsonl').open('a') as log:
                log.write(json.dumps({'path': self.path, 'at': time.time()}) + '\n')
        body = json.loads(self.rfile.read(int(self.headers.get('Content-Length', '0'))))
        control = ROOT / 'mock-control.json'
        cfg = json.loads(control.read_text()) if control.exists() else {}
        (ROOT / 'mock-seen.json').write_text(json.dumps({'path': self.path, 'at': time.time()}))
        time.sleep(cfg.get('delay', 0))
        if cfg.get('error'):
            self.send_response(cfg['error'])
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"error":{"type":"invalid_request_error","message":"local injected failure"}}')
            return
        model = body.get('model', 'gpt-6-luna')
        rid = 'mock_' + uuid.uuid4().hex
        # OpenAI/Gemini input includes cache; Anthropic input excludes cache.
        inp, cache, outp = cfg.get('input', 1000), cfg.get('cache', 200), cfg.get('output', 500)
        text = '本地模拟上游：晶能已点亮。'
        usage = {'input_tokens': inp, 'output_tokens': outp, 'total_tokens': inp + outp,
                 'input_tokens_details': {'cached_tokens': cache}}
        out = {'id': rid, 'object': 'response', 'created_at': int(time.time()), 'status': 'completed',
               'model': model, 'output': [{'id': 'msg_' + rid, 'type': 'message', 'status': 'completed',
               'role': 'assistant', 'content': [{'type': 'output_text', 'text': text, 'annotations': []}]}], 'usage': usage}
        if cfg.get('search'):
            out['output'].insert(0, {'type': 'web_search_call', 'id': 'search_' + rid, 'status': 'completed', 'action': {'type': 'search', 'query': 'test'}})
        if '/chat/completions' in self.path:
            out = {'id': rid, 'object': 'chat.completion', 'created': int(time.time()), 'model': model,
                   'choices': [{'index': 0, 'message': {'role': 'assistant', 'content': text}, 'finish_reason': 'stop'}],
                   'usage': {'prompt_tokens': inp, 'completion_tokens': outp, 'total_tokens': inp + outp,
                             'prompt_tokens_details': {'cached_tokens': cache}}}
        elif '/messages' in self.path:
            out = {'id': rid, 'type': 'message', 'role': 'assistant', 'model': model,
                   'content': [{'type': 'text', 'text': text}], 'stop_reason': 'end_turn',
                   'usage': {'input_tokens': inp - cache, 'output_tokens': outp, 'cache_read_input_tokens': cache, 'cache_creation_input_tokens': 0}}
        elif ':generateContent' in self.path or ':streamGenerateContent' in self.path:
            out = {'candidates': [{'content': {'parts': [{'text': text}], 'role': 'model'}, 'finishReason': 'STOP', 'index': 0}],
                   'usageMetadata': {'promptTokenCount': inp, 'candidatesTokenCount': outp,
                                     'cachedContentTokenCount': cache, 'totalTokenCount': inp + outp}, 'modelVersion': model}
        self.send_response(200)
        self.send_header('x-request-id', rid)
        stream = body.get('stream') or ':streamGenerateContent' in self.path
        self.send_header('Content-Type', 'text/event-stream' if stream else 'application/json')
        self.end_headers()
        if not stream:
            self.wfile.write(json.dumps(out).encode())
            return

        def event(data, name=None):
            self.wfile.write(((('event: ' + name + '\n') if name else '') + 'data: ' + json.dumps(data, ensure_ascii=False) + '\n\n').encode())
            self.wfile.flush()

        # A cutoff before the first event means the gateway has no valid
        # provider usage to settle; it must release reservation and leave cards
        # and balance untouched.
        if cfg.get('cutoff'):
            return
        if '/messages' in self.path:
            event({'type': 'message_start', 'message': {**out, 'content': [], 'usage': {**out['usage'], 'output_tokens': 0}}}, 'message_start')
            event({'type': 'content_block_start', 'index': 0, 'content_block': {'type': 'text', 'text': ''}}, 'content_block_start')
            event({'type': 'content_block_delta', 'index': 0, 'delta': {'type': 'text_delta', 'text': text}}, 'content_block_delta')
            event({'type': 'content_block_stop', 'index': 0}, 'content_block_stop')
            event({'type': 'message_delta', 'delta': {'stop_reason': 'end_turn'}, 'usage': out['usage']}, 'message_delta')
            event({'type': 'message_stop'}, 'message_stop')
        elif 'GenerateContent' in self.path:
            event(out)
        elif '/chat/completions' in self.path:
            event({**out, 'object': 'chat.completion.chunk', 'choices': [{'index': 0, 'delta': {'role': 'assistant', 'content': text}, 'finish_reason': None}]})
            event({**out, 'object': 'chat.completion.chunk', 'choices': [{'index': 0, 'delta': {}, 'finish_reason': 'stop'}]})
            self.wfile.write(b'data: [DONE]\n\n')
        else:
            event({'type': 'response.created', 'response': {**out, 'output': [], 'status': 'in_progress', 'usage': None}}, 'response.created')
            event({'type': 'response.output_text.delta', 'item_id': 'msg_' + rid, 'output_index': 0, 'content_index': 0, 'delta': text}, 'response.output_text.delta')
            event({'type': 'response.completed', 'response': out}, 'response.completed')


if __name__ == '__main__':
    ThreadingHTTPServer(('127.0.0.1', 19086), Handler).serve_forever()
