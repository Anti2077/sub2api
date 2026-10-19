import type { ApiConfig, CodedError, ErrorCode } from './types'

type Json = null | boolean | number | string | Json[] | { [key: string]: Json }
type JsonObject = { [key: string]: Json }

export interface ProbeUsage {
  /** Input tokens the upstream billed for the request, including cached prompt tokens. */
  inputTokens: number
  responseModel?: string
}

const coded = (message: string, code: ErrorCode, extra?: Partial<CodedError>): CodedError => Object.assign(new Error(message), { code }, extra)
const object = (value: Json | undefined): JsonObject => value !== null && typeof value === 'object' && !Array.isArray(value) ? value : {}
const count = (value: Json | undefined) => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined
const text = (value: Json | undefined) => typeof value === 'string' ? value : ''

/**
 * Chat Completions reports `prompt_tokens`, which already includes cached tokens. Responses reports
 * `input_tokens` the same way. Messages reports cache writes and cache reads outside `input_tokens`.
 */
export function inputTokens(usage: Json | undefined, format: ApiConfig['format']): number | undefined {
  const value = object(usage)
  if (format === 'openai') return count(value.prompt_tokens)
  const base = count(value.input_tokens)
  if (format === 'responses' || base === undefined) return base
  return base + (count(value.cache_creation_input_tokens) ?? 0) + (count(value.cache_read_input_tokens) ?? 0)
}

function parse(payload: string): JsonObject {
  try { return object(JSON.parse(payload) as Json) } catch { throw coded('The API response is not valid JSON.', 'bad_stream_json') }
}

/**
 * The HTTP status is the error; a body that is not JSON is kept as plain text instead of masking the status. The
 * proxy marks an upstream that answered with a web page instead of an API reply.
 */
function httpError(status: number, body: string): CodedError {
  const trimmed = body.trim()
  let data: JsonObject = {}
  if (trimmed.startsWith('{')) {
    try { data = object(JSON.parse(trimmed) as Json) } catch { data = {} }
  }
  const error = object(data.error)
  const detail = text(error.message) || text(data.message) || trimmed.slice(0, 300)
  return coded(`HTTP ${status}${detail ? `: ${detail}` : ''}`, error.code === 'upstream_not_api' ? 'upstream_not_api' : 'http', { httpStatus: status })
}

/**
 * Reads the usage report of a JSON or SSE response. The generated text is ignored. A stream ends at the protocol's
 * terminal event once usage is known, since some upstreams keep the connection open after it.
 */
export async function readUsage(response: Response, format: ApiConfig['format']): Promise<ProbeUsage> {
  if (!response.ok) throw httpError(response.status, await response.text())
  const usage: JsonObject = {}
  let responseModel: string | undefined, terminal = false
  // Later events may repeat a field as null, such as Messages `message_delta`; only counts replace earlier values.
  const merge = (value: Json | undefined) => {
    for (const [key, field] of Object.entries(object(value))) if (count(field) !== undefined) usage[key] = field
  }
  const accept = (data: JsonObject) => {
    if (data.error || data.type === 'error') throw coded(text(object(data.error).message) || text(data.message) || 'The API stream sent an error event.', 'upstream_stream_error')
    const response = object(data.response), message = object(data.message)
    // Responses reports a failed generation inside the response object, usually with null usage.
    if (format === 'responses' && (data.type === 'response.failed' || response.status === 'failed')) {
      throw coded(text(object(response.error).message) || 'The API reported a failed response.', 'upstream_stream_error')
    }
    responseModel = text(data.model) || text(response.model) || text(message.model) || responseModel
    if (format === 'anthropic') { merge(message.usage); if (data.type !== 'message_start') merge(data.usage) }
    else if (format === 'responses') merge(data.usage ?? response.usage)
    else if (data.usage) merge(data.usage)
    terminal ||= data.type === 'message_stop' || data.type === 'response.completed' || data.type === 'response.incomplete'
  }
  if (!response.headers.get('content-type')?.includes('text/event-stream')) accept(parse(await response.text()))
  else {
    if (!response.body) throw coded('The API stream has no body.', 'no_stream_body')
    const reader = response.body.getReader(), decoder = new TextDecoder()
    let buffer = ''
    const frame = (value: string) => {
      const payload = value.split(/\r?\n/).filter(line => line.startsWith('data:')).map(line => line.slice(5).replace(/^ /, '')).join('\n')
      if (payload.trim() === '[DONE]') terminal = true
      else if (payload) accept(parse(payload))
    }
    const settled = () => terminal && inputTokens(usage, format) !== undefined
    try {
      while (!settled()) {
        const { done, value } = await reader.read()
        buffer += done ? decoder.decode() : decoder.decode(value, { stream: true })
        let match: RegExpExecArray | null
        while (!settled() && (match = /\r?\n\r?\n/.exec(buffer))) { frame(buffer.slice(0, match.index)); buffer = buffer.slice(match.index + match[0].length) }
        if (done) { if (!settled() && buffer.trim()) frame(buffer); break }
      }
    } finally { void reader.cancel().catch(() => {}); reader.releaseLock() }
  }
  const tokens = inputTokens(usage, format)
  if (tokens === undefined) throw coded('The API response has no input token count in usage. The tokenizer probe needs this count.', 'no_usage')
  return { inputTokens: tokens, responseModel }
}
